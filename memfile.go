package syzgydb

import (
	"encoding/binary"
	"errors"
	"log"
	"os"

	"github.com/go-mmap/mmap"
)

type FileMode int

const (
	CreateIfNotExists  FileMode = 0 // Create the file only if it doesn't exist
	ReadWrite          FileMode = 1 // Open the file for read/write access
	ReadOnly           FileMode = 2 // Open the file for read-only access
	CreateAndOverwrite FileMode = 3 // Always create and overwrite the file if it exists
)

// The memory file consists of a header followed by a series of records.
// Each record is:
// uint64 - total length of record
// uint64 - ID, or deleted is all 0xffffffffffffffff

const growthPercentage = 0.05
const minGrowthBytes = 4096
const deletedRecordMarker = 0xffffffffffffffff

type memfile struct {
	*mmap.File
	// Header size which we ignore
	headerSize int64

	// offsets of each record id into the file
	idOffsets map[uint64]int64

	freemap freeMap

	name string
}

/*
deleteRecord marks a record as deleted and frees its space.

Parameters:
- id: The ID of the record to delete.

Returns:
- An error if the record is not found.
*/
func (mf *memfile) deleteRecord(id uint64) error {

	// Check if the record ID exists
	offset, exists := mf.idOffsets[id]
	if !exists {
		return errors.New("record not found")
	}

	// Mark the record as deleted
	mf.writeUint64(offset+8, deletedRecordMarker)

	// Mark the space as free
	recordLength := mf.readUint64(offset)
	mf.freemap.markFree(int(offset), int(recordLength))

	// Remove the record ID from the idOffsets map
	delete(mf.idOffsets, id)

	return nil
}

/*
createMemFile creates a new memory-mapped file and writes the header if the file is new.

Parameters:
- name: The name of the file.
- header: The header data to write if the file is new.

Returns:
- A pointer to the created memfile.
- An error if the file cannot be created.
*/
func createMemFile(name string, header []byte, mode FileMode) (*memfile, error) {
	var flags int

	// Set flags based on the mode
	switch mode {
	case ReadOnly:
		flags = os.O_RDONLY
	case ReadWrite:
		flags = os.O_RDWR
	case CreateIfNotExists:
		flags = os.O_CREATE | os.O_RDWR
	case CreateAndOverwrite:
		flags = os.O_CREATE | os.O_TRUNC | os.O_RDWR
	default:
		return nil, errors.New("invalid mode")
	}

	// Open the file with the specified flags
	file, err := os.OpenFile(name, flags, 0644)
	if err != nil {
		return nil, err
	}
	file.Close()

	// Open the file with mmap
	f, err := mmap.OpenFile(name, mmap.Read|mmap.Write)
	if err != nil {
		return nil, err
	}

	ret := &memfile{
		File:       f,
		idOffsets:  make(map[uint64]int64),
		headerSize: int64(len(header)),
		name:       name,
	}

	// Check if the file is new by checking its size
	if ret.Len() == 0 {
		ret.ensureLength(len(header))
		// Write the header to the file
		if _, err := ret.WriteAt(header, 0); err != nil {
			return nil, err
		}
		ret.Sync()
	} else {
		// Process existing records
		offset := ret.headerSize
		for offset < int64(ret.Len()) {
			recordLength := ret.readUint64(offset)
			if recordLength == 0 {
				// Mark the remaining space in the file as free
				remainingLength := int(int64(ret.Len()) - offset)
				ret.freemap.markFree(int(offset), remainingLength)
				break // End of valid data
			}

			id := ret.readUint64(offset + 8)
			if id == deletedRecordMarker {
				// Record is marked as deleted, add to freemap
				ret.freemap.markFree(int(offset), int(recordLength))
			} else {
				// Record is valid, add to idOffsets
				ret.idOffsets[id] = offset
			}

			offset += int64(recordLength)
		}
	}

	return ret, nil
}

/*
ensureLength checks if the file is at least the given length, and if not,
extends it and remaps the file.

Parameters:
- length: The minimum length the file should be.
*/
func (mf *memfile) ensureLength(length int) {
	curSize := mf.File.Len()

	if curSize >= length {
		return
	}

	log.Printf("Current file size: %d bytes, requested length: %d bytes\n", curSize, length)

	// Calculate the growth size
	growthSize := length - curSize
	growthSize = max(growthSize, minGrowthBytes)
	growthSize = max(growthSize, int(float64(curSize)*growthPercentage))

	length += growthSize

	if curSize >= length {
		return
	}

	log.Printf("Extending file from %d to %d bytes\n", curSize, length)

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

	if curSize < int(mf.headerSize) {
		curSize = int(mf.headerSize)
	}

	// Update freemap with the extended range
	mf.freemap.markFree(int(curSize), length-int(curSize))

	// Re-obtain the memory-mapped file
	mf.File, err = mmap.OpenFile(mf.name, mmap.Read|mmap.Write)
	if err != nil {
		log.Panic(err)
	}
}

/*
addRecord adds a new record to the memory-mapped file.

Parameters:
- id: The ID of the record.
- data: The data to be stored in the record.
*/
func (mf *memfile) addRecord(id uint64, data []byte) bool {
	// Calculate the total length of the record
	recordLength := 16 + len(data) // 8 bytes for length, 8 bytes for ID

	// Determine if the record was newly added or updated
	wasNew := true

	// Find a free location for the new record
	start, remaining, err := mf.freemap.getFreeRange(recordLength)
	if err != nil {
		log.Printf("No free space found, ensuring file length for record ID %d\n", id)
		mf.ensureLength(mf.File.Len() + recordLength)
		start, remaining, err = mf.freemap.getFreeRange(recordLength)
		if err != nil {
			log.Panic("Failed to allocate space for the new record")
		}
	}

	//log.Printf("Adding record ID %d at offset %d with length %d\n", id, start, recordLength)

	// Adjust the record length if the remaining space is 16 bytes or less
	if remaining > 0 && remaining <= 16 {
		mf.freemap.markUsed(int(start)+recordLength, int(remaining))
		recordLength += int(remaining)
		remaining = 0
	}

	// Write the record to the file with the adjusted length
	offset := start
	mf.writeUint64(offset, uint64(recordLength))

	// Write the ID
	mf.writeUint64(offset+8, id)

	// Write the data
	mf.WriteAt(data, offset+16)

	// If there is remaining space greater than 16 bytes, mark it as a deleted record
	if remaining > 16 {
		mf.writeUint64(offset+int64(recordLength), uint64(remaining))
		mf.writeUint64(offset+int64(recordLength)+8, deletedRecordMarker)
		mf.freemap.markFree(int(start)+recordLength, int(remaining))
	}

	// If the record already existed, mark the old space as free
	if oldOffset, exists := mf.idOffsets[id]; exists {
		oldLength := mf.readUint64(oldOffset)
		mf.writeUint64(oldOffset+8, deletedRecordMarker)
		mf.freemap.markFree(int(oldOffset), int(oldLength))
		log.Printf("Updated existing record ID %d, freeing old space at offset %d\n", id, oldOffset)
		wasNew = false
	}

	// Update the idOffsets map
	mf.idOffsets[id] = int64(offset)

	return wasNew
}

/*
readUint64 reads an unsigned 64-bit integer from the specified offset.

Parameters:
- offset: The offset from which to read.

Returns:
- The unsigned 64-bit integer read from the file.
*/
func (mf *memfile) readUint64(offset int64) uint64 {
	// Read 8 bytes from the specified offset
	buf := make([]byte, 8)
	mf.ReadAt(buf, offset)
	return binary.BigEndian.Uint64(buf)
}

/*
readRecord reads a record by its ID.

Parameters:
- id: The ID of the record to read.

Returns:
- The data of the record.
- An error if the record is not found.
*/
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

/*
func (mf *memfile) writeByte(offset int64, value byte) {
	mf.WriteAt([]byte{value}, offset)
}

func (mf *memfile) writeUint32(offset int64, value uint32) {
	// Convert value to a byte slice
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, value)
	mf.WriteAt(buf, offset)
}

func (mf *memfile) readUint32(offset int64) uint32 {
	// Read 4 bytes from the specified offset
	buf := make([]byte, 4)
	mf.ReadAt(buf, offset)
	return binary.BigEndian.Uint32(buf)
}*/

/*
writeUint64 writes an unsigned 64-bit integer to the specified offset.

Parameters:
- offset: The offset at which to write.
- value: The unsigned 64-bit integer to write.
*/
func (mf *memfile) writeUint64(offset int64, value uint64) {
	// use mf.File.WriteByte() to write the value to the file
	// assume that it is already large enough.

	// convert value to a byte slice
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, value)
	mf.WriteAt(buf, int64(offset))
}
