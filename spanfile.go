package syzgydb

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	activeMagic = 0x5350414E // 'SPAN'
	freeMagic   = 0x46524545 // 'FREE'
)

type DataStream struct {
	StreamID uint8
	Data     []byte
}

type Span struct {
	MagicNumber    uint32
	Length         uint64
	SequenceNumber uint64
	RecordID       string
	DataStreams    []DataStream
	Checksum       [32]byte
}

type IndexEntry struct {
	Offset         uint64
	Span           *Span
	SequenceNumber uint64
}

type DB struct {
	file           *os.File
	mmapData       []byte
	index          map[string]IndexEntry
	freeList       []FreeSpan
	sequenceNumber uint64
	fileMutex      sync.Mutex
}

type FreeSpan struct {
	Offset uint64
	Length uint64
}

type OpenOptions struct {
	CreateIfNotExists bool
	OverwriteExisting bool
}

func OpenFile(filename string, options OpenOptions) (*DB, error) {
	flags := os.O_RDWR
	if options.CreateIfNotExists {
		flags |= os.O_CREATE
	}
	if options.OverwriteExisting {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return nil, err
	}

	mmapData, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	db := &DB{
		file:           file,
		mmapData:       mmapData,
		index:          make(map[string]IndexEntry),
		freeList:       []FreeSpan{},
		sequenceNumber: 0,
	}

	err = db.scanFile()
	if err != nil {
		mmapData.Unmap()
		file.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) scanFile() error {
	offset := uint64(0)
	fileSize := uint64(len(db.mmapData))
	highestSeqNum := uint64(0)

	for offset < fileSize {
		magicNumber := binary.BigEndian.Uint32(db.mmapData[offset : offset+4])
		length := binary.BigEndian.Uint64(db.mmapData[offset+4 : offset+12])
		spanData := db.mmapData[offset : offset+length]

		if !verifyChecksum(spanData) {
			offset += length
			continue
		}

		span, err := parseSpan(spanData)
		if err != nil {
			offset += length
			continue
		}

		if span.SequenceNumber > highestSeqNum {
			highestSeqNum = span.SequenceNumber
		}

		if magicNumber == activeMagic {
			existingEntry, exists := db.index[span.RecordID]
			if !exists || span.SequenceNumber > existingEntry.SequenceNumber {
				db.index[span.RecordID] = IndexEntry{
					Offset:         offset,
					Span:           span,
					SequenceNumber: span.SequenceNumber,
				}
			}
		} else if magicNumber == freeMagic {
			db.addFreeSpan(offset, length)
		}

		offset += length
	}

	db.sequenceNumber = highestSeqNum + 1
	return nil
}

func (db *DB) addFreeSpan(offset, length uint64) {
	db.freeList = append(db.freeList, FreeSpan{
		Offset: offset,
		Length: length,
	})
	// Coalescing logic can be added here if needed
}

func (db *DB) WriteRecord(recordID string, dataStreams []DataStream) error {
	db.fileMutex.Lock()
	defer db.fileMutex.Unlock()

	sequenceNumber := db.sequenceNumber
	db.sequenceNumber++

	span := &Span{
		MagicNumber:    activeMagic,
		SequenceNumber: sequenceNumber,
		RecordID:       recordID,
		DataStreams:    dataStreams,
	}

	spanBytes, err := serializeSpan(span)
	if err != nil {
		return err
	}

	checksum := calculateChecksum(spanBytes)
	span.Checksum = checksum
	spanBytes = append(spanBytes, checksum[:]...)

	offset, err := db.allocateSpan(len(spanBytes))
	if err != nil {
		return err
	}

	err = db.writeAt(spanBytes, offset)
	if err != nil {
		return err
	}

	db.index[recordID] = IndexEntry{
		Offset:         offset,
		Span:           span,
		SequenceNumber: sequenceNumber,
	}

	if oldEntry, exists := db.index[recordID]; exists && oldEntry.SequenceNumber < sequenceNumber {
		err = db.markSpanAsFreed(oldEntry.Offset)
		if err != nil {
			return err
		}
		db.addFreeSpan(oldEntry.Offset, oldEntry.Span.Length)
	}

	return nil
}

func (db *DB) allocateSpan(size int) (uint64, error) {
	for i, freeSpan := range db.freeList {
		if freeSpan.Length >= uint64(size) {
			offset := freeSpan.Offset

			if freeSpan.Length > uint64(size) {
				db.freeList[i].Offset += uint64(size)
				db.freeList[i].Length -= uint64(size)
			} else {
				db.freeList = append(db.freeList[:i], db.freeList[i+1:]...)
			}

			return offset, nil
		}
	}

	offset := uint64(len(db.mmapData))
	err := db.appendToFile(make([]byte, size))
	if err != nil {
		return 0, err
	}

	return offset, nil
}

func (db *DB) writeAt(data []byte, offset uint64) error {
	if offset+uint64(len(data)) > uint64(len(db.mmapData)) {
		err := db.file.Truncate(int64(offset + uint64(len(data))))
		if err != nil {
			return err
		}
		db.mmapData, err = mmap.Map(db.file, mmap.RDWR, 0)
		if err != nil {
			return err
		}
	}

	copy(db.mmapData[offset:], data)
	return msync(db.mmapData[offset : offset+uint64(len(data))])
}

func (db *DB) markSpanAsFreed(offset uint64) error {
	binary.BigEndian.PutUint32(db.mmapData[offset:offset+4], freeMagic)
	return msync(db.mmapData[offset : offset+4])
}

func (db *DB) ReadRecord(recordID string) (*Span, error) {
	entry, exists := db.index[recordID]
	if !exists {
		return nil, fmt.Errorf("record not found")
	}
	return entry.Span, nil
}

func (db *DB) IterateRecords(callback func(recordID string, dataStreams []DataStream) error) error {
	for recordID, entry := range db.index {
		err := callback(recordID, entry.Span.DataStreams)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetStats() (size uint64, numRecords int) {
	size = uint64(len(db.mmapData))
	numRecords = len(db.index)
	return
}

func (db *DB) DumpFile(output io.Writer) error {
	offset := uint64(0)
	fileSize := uint64(len(db.mmapData))
	for offset < fileSize {
		span, err := parseSpanAtOffset(db.mmapData, offset)
		if err != nil {
			fmt.Fprintf(output, "Error parsing span at offset %d: %v\n", offset, err)
			break
		}

		fmt.Fprintf(output, "Offset: %d\n", offset)
		fmt.Fprintf(output, "Magic Number: %s\n", magicNumberToString(span.MagicNumber))
		fmt.Fprintf(output, "Length: %d bytes\n", span.Length)
		fmt.Fprintf(output, "Sequence Number: %d\n", span.SequenceNumber)
		fmt.Fprintf(output, "Record ID: %s\n", span.RecordID)
		fmt.Fprintf(output, "Data Streams:\n")
		for _, ds := range span.DataStreams {
			fmt.Fprintf(output, "  Stream ID: %d, Length: %d bytes\n", ds.StreamID, len(ds.Data))
		}
		fmt.Fprintf(output, "Checksum: %x\n", span.Checksum)
		fmt.Fprintln(output)

		offset += span.Length
	}
	return nil
}

func serializeSpan(span *Span) ([]byte, error) {
	// Implement span serialization
	// This is a placeholder implementation
	return nil, nil
}

func parseSpan(data []byte) (*Span, error) {
	// Implement span parsing
	// This is a placeholder implementation
	return nil, nil
}

func parseSpanAtOffset(data []byte, offset uint64) (*Span, error) {
	// Implement parsing logic at a specific offset
	// This is a placeholder implementation
	return nil, nil
}

func calculateChecksum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func verifyChecksum(data []byte) bool {
	// Implement checksum verification
	// This is a placeholder implementation
	return false
}

func appendToFile(data []byte) error {
	// Implement file appending
	// This is a placeholder implementation
	return nil
}

func msync(data []byte) error {
	// Implement msync logic
	// This is a placeholder implementation
	return nil
}

func mmap(file *os.File) ([]byte, error) {
	// Implement mmap logic
	// This is a placeholder implementation
	return nil, nil
}

func magicNumberToString(magic uint32) string {
	// Convert magic number to string
	// This is a placeholder implementation
	return ""
}
