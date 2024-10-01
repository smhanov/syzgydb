package main

import (
	"log"
	"os"

	"github.com/go-mmap/mmap"
)

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
func (mf *memfile) ensureLength(length int) error {
	if mf.File.Len() >= length {
		return nil
	}

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

	// Check the current file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Panic(err)
	}

	// Check if the file is already the given length
	if fileInfo.Size() >= int64(length) {
		return nil
	}

	// Increase the file size
	if err := file.Truncate(int64(length)); err != nil {
		log.Panic(err)
	}

	// Update freemap with the extended range
	mf.freemap.markFree(int(fileInfo.Size()), length-int(fileInfo.Size()))

	// Re-obtain the memory-mapped file
	mf.File, err = mmap.OpenFile(mf.name, mmap.Read|mmap.Write)
	if err != nil {
		return err
	}

	return nil
}
