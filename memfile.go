package main

import (
	"log"
	"os"

	"github.com/go-mmap/mmap"
)

// The memory file consists of a header followed by a series of records.
// Each record is:
// uint64 - total length of record
// uint64 - ID, or deleted is all 0xffffffffffffffff

type memfile struct {
	*mmap.File
	//Header size which we ignore
	headerSize int

	// offsets of each record id into the file
	idOffsets map[uint64]int64

	freemap FreeMap

	name string
}

func createMemFile(name string, headerSize int) (*memfile, error) {
	f, err := mmap.OpenFile(name, mmap.Read|mmap.Write)
	if err != nil {
		return nil, err
	}
	ret := &memfile{File: f,
		headerSize: headerSize,
		name:       name,
	}

	return ret, nil
}

// check if the file is at least the given length, and if not, extend it
// and remap the file
func (mf *memfile) ensureLength(length int) {
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
	// use copy-on-write semantics
	// first, use the freemap to find a free location for the new or updated record

	// if there was no free space, ensure the file is large enough using ensureLength

	// write the record to the file. The record
	// is the length of the record, the id, and the data

	// sync the file to disk

	// if the record already existed, then mark it as deleted by writing
	// 0xffffffffffffffff as the id in the old location
	// read its old length and mark that space as free
}
