package main

import "github.com/go-mmap/mmap"

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
	return nil
}
