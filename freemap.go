package syzgydb

import (
	"errors"
	"fmt"
	"log"
	"sort"
)

const verboseFreeMap = false

// markUsed marks a range of space as used.
func (fm *FreeMap) markUsed(start, length int) {
	if verboseFreeMap {
		log.Printf("markUsed: start=%d, length=%d\n", start, length)
	}
	if length <= 0 {
		return
	}

	for i, s := range fm.freeSpaces {
		if s.start <= start && start+length <= s.start+s.length {
			// Adjust the free space
			if start == s.start {
				// Used space is at the beginning
				fm.freeSpaces[i].start += length
				fm.freeSpaces[i].length -= length
			} else if start+length == s.start+s.length {
				// Used space is at the end
				fm.freeSpaces[i].length -= length
			} else {
				// Used space is in the middle, split the free space
				fm.freeSpaces = append(fm.freeSpaces, space{
					start:  start + length,
					length: s.start + s.length - (start + length),
				})
				fm.freeSpaces[i].length = start - s.start
			}

			// Remove the free space if its length is zero
			if fm.freeSpaces[i].length == 0 {
				fm.freeSpaces = append(fm.freeSpaces[:i], fm.freeSpaces[i+1:]...)
			}
			break
		}
	}

	if verboseFreeMap {
		fm.logSpaces()
	}
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
	if verboseFreeMap {
		log.Printf("markFree: start=%d, length=%d\n", start, length)
	}
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

	if verboseFreeMap {
		fm.logSpaces()
	}
}

// getFreeRange finds a free range of the specified length and marks it as used.
func (fm *FreeMap) getFreeRange(length int) (int64, int64, error) {
	if verboseFreeMap {
		log.Printf("getFreeRange: length=%d\n", length)
	}
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
			if verboseFreeMap {
				fm.logSpaces()
			}
			return int64(start), int64(remaining), nil
		}
	}

	return 0, 0, errors.New("no sufficient free space available")
}

// logSpaces logs all the free ranges in the FreeMap.
func (fm *FreeMap) logSpaces() {
	fmt.Println("Free spaces:")
	for _, s := range fm.freeSpaces {
		fmt.Printf("Start: %d, Length: %d\n", s.start, s.length)
	}
}
