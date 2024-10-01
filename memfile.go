package main

import (
	"encoding/binary"
	"errors"
	"log"
	"os"
	"sync"

	"github.com/go-mmap/mmap"
)

// The memory file consists of a header followed by a series of records.
// Each record is:
// uint64 - total length of record
// uint64 - ID, or deleted is all 0xffffffffffffffff

const debug = true

type memfile struct {
	*mmap.File
	sync.Mutex
	// Header size which we ignore
	headerSize int64

	// offsets of each record id into the file
	idOffsets map[uint64]int64

	freemap FreeMap

	name string
}

func (mf *memfile) deleteRecord(id uint64) error {
	mf.Lock()
	defer mf.Unlock()

	mf.Lock()
	defer mf.Unlock()

	// Check if the record ID exists
	offset, exists := mf.idOffsets[id]
	if !exists {
		return errors.New("record not found")
	}

	// Mark the record as deleted
	mf.writeUint64(offset+8, 0xffffffffffffffff)

	// Mark the space as free
	recordLength := mf.readUint64(offset)
	mf.freemap.markFree(int(offset), int(recordLength))

	// Remove the record ID from the idOffsets map
	delete(mf.idOffsets, id)

	return nil
}

func createMemFile(name string, headerSize int64) (*memfile, error) {
	f, err := mmap.OpenFile(name, mmap.Read|mmap.Write)
	if err != nil {
		return nil, err
	}
	ret := &memfile{
		File:       f,
		idOffsets:  make(map[uint64]int64),
		headerSize: headerSize,
		name:       name,
	}

	return ret, nil
}

// check if the file is at least the given length, and if not, extend it
// and remap the file
func (mf *memfile) ensureLength(length int) {
	mf.Lock()
	defer mf.Unlock()

	curSize := mf.File.Len()
	if curSize >= length {
		return
	}

	length += 4096

	// Close the current memory-mapped file
	if err := mf.File.Close(); err != nil {
		log.Panic(err)
	}

	// Open the file on disk
	file, err := os.OpenFile(mf.name, os.O_RDWR, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()

	// Increase the file size
	if err := file.Truncate(int64(length)); err != nil {
		log.Panic(err)
	}

	// Update freemap with the extended range
	mf.freemap.markFree(int(curSize), length-int(curSize))

	// Re-obtain the memory-mapped file
	mf.File, err = mmap.OpenFile(mf.name, mmap.Read|mmap.Write)
	if err != nil {
		log.Panic(err)
	}
}

func (mf *memfile) addRecord(id uint64, data []byte) {
	mf.Lock()
	defer mf.Unlock()

	// Calculate the total length of the record
	recordLength := 16 + len(data) // 8 bytes for length, 8 bytes for ID

	// Find a free location for the new record
	start, err := mf.freemap.getFreeRange(recordLength)
	if err != nil {
		// If no free space, ensure the file is large enough
		mf.ensureLength(mf.File.Len() + recordLength)
		start, err = mf.freemap.getFreeRange(recordLength)
		if err != nil {
			log.Panic("Failed to allocate space for the new record")
		}
	}

	// Write the record to the file
	offset := start + mf.headerSize
	mf.writeUint64(offset, uint64(recordLength))

	// Write the ID
	mf.writeUint64(offset+8, id)

	// Write the data
	mf.WriteAt(data, offset+16)

	// If the record already existed, mark the old space as free
	if oldOffset, exists := mf.idOffsets[id]; exists {
		mf.writeUint64(oldOffset, 0xffffffffffffffff)
		oldLength := mf.readUint64((oldOffset))
		mf.freemap.markFree(int(oldOffset), int(oldLength))
	}

	// Update the idOffsets map
	mf.idOffsets[id] = int64(offset)
}

func (mf *memfile) readUint64(offset int64) uint64 {
	mf.Lock()
	defer mf.Unlock()

	// Read 8 bytes from the specified offset
	buf := make([]byte, 8)
	mf.ReadAt(buf, offset)
	return binary.LittleEndian.Uint64(buf)
}

func (mf *memfile) readRecord(id uint64) ([]byte, error) {
	// Check if the record ID exists
	offset, exists := mf.idOffsets[id]
	if !exists {
		return nil, errors.New("record not found")
	}

	// Read the total length of the record
	recordLength := mf.readUint64(offset)

	// Read the record data
	data := make([]byte, recordLength-16) // Subtract 16 bytes for length and ID
	mf.ReadAt(data, offset+16)

	return data, nil
}

func (mf *memfile) writeUint64(offset int64, value uint64) {
	mf.Lock()
	defer mf.Unlock()

	// use mf.File.WriteByte() to write the value to the file
	// assume that it is already large enough.

	// convert value to a byte slice
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, value)
	mf.WriteAt(buf, int64(offset))
}
