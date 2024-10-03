package syzgydb

import (
	"errors"
	"sort"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type FreeMap struct {
	freeSpaces []space
}

type space struct {
	start  int
	length int
}

// markFree marks a range of space as free.
func (fm *FreeMap) markFree(start, length int) {
	if length <= 0 {
		return
	}

	// Add the new free space
	fm.freeSpaces = append(fm.freeSpaces, space{start, length})

	// Sort the free spaces by start position
	sort.Slice(fm.freeSpaces, func(i, j int) bool {
		return fm.freeSpaces[i].start < fm.freeSpaces[j].start
	})

	// Merge contiguous free spaces
	merged := []space{}
	for _, s := range fm.freeSpaces {
		if len(merged) == 0 || merged[len(merged)-1].start+merged[len(merged)-1].length < s.start {
			merged = append(merged, s)
		} else {
			merged[len(merged)-1].length = max(merged[len(merged)-1].start+merged[len(merged)-1].length, s.start+s.length) - merged[len(merged)-1].start
		}
	}
	fm.freeSpaces = merged
}

// getFreeRange finds a free range of the specified length and marks it as used.
func (fm *FreeMap) getFreeRange(length int) (int64, int64, error) {
	if length <= 0 {
		return 0, 0, errors.New("length must be positive")
	}

	for i, s := range fm.freeSpaces {
		if s.length >= length {
			start := s.start
			fm.freeSpaces[i].start += length
			fm.freeSpaces[i].length -= length

			if fm.freeSpaces[i].length == 0 {
				fm.freeSpaces = append(fm.freeSpaces[:i], fm.freeSpaces[i+1:]...)
			}

			remaining := s.length - length
			return int64(start), int64(remaining), nil
		}
	}

	return 0, 0, errors.New("no sufficient free space available")
}

// dump prints all the free ranges in the FreeMap.
/*
func (fm *FreeMap) dump() {
	for _, s := range fm.freeSpaces {
		fmt.Printf("Start: %d, Length: %d\n", s.start, s.length)
	}
}*/
