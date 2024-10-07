package syzgydb

import (
	"testing"
)

func TestMarkFreeAndGetFreeRange(t *testing.T) {
	fm := &freeMap{}

	// Mark some spaces as free
	fm.markFree(0, 10)
	fm.markFree(20, 5)
	fm.markFree(15, 5)

	// Test merging of contiguous spaces
	if len(fm.freeSpaces) != 2 {
		t.Errorf("Expected 2 free spaces, got %d", len(fm.freeSpaces))
	}

	// Test getting a free range
	var remaining int64
	start, _, err := fm.getFreeRange(5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 0 {
		t.Errorf("Expected start to be 0, got %d", start)
	}

	// Test getting another free range
	start, remaining, err = fm.getFreeRange(10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 15 {
		t.Errorf("Expected start to be 15, got %d", start)
	}

	// Test insufficient space
	_, _, err = fm.getFreeRange(10)
	if err == nil {
		t.Errorf("Expected error due to insufficient space, got nil")
	}
	// Test remaining space
	if remaining != 0 {
		t.Errorf("Expected remaining to be 0, got %d", remaining)
	}

	// Test getting a free range with remaining space
	fm.markFree(30, 20)
	start, remaining, err = fm.getFreeRange(10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 30 {
		t.Errorf("Expected start to be 30, got %d", start)
	}
	if remaining != 10 {
		t.Errorf("Expected remaining to be 10, got %d", remaining)
	}
}

func TestMergeContiguousFreeSpaces(t *testing.T) {
	fm := &freeMap{}

	// Mark the specified free spaces
	fm.markFree(15, 468)
	fm.markFree(535, 3576)
	fm.markFree(483, 52)

	// Verify that there is now one free region at start 15 of length 4096
	if len(fm.freeSpaces) != 1 {
		t.Errorf("Expected 1 free space, got %d", len(fm.freeSpaces))
	}

	expectedStart := 15
	expectedLength := 4096
	if fm.freeSpaces[0].start != expectedStart || fm.freeSpaces[0].length != expectedLength {
		t.Errorf("Expected free space at start %d with length %d, got start %d with length %d",
			expectedStart, expectedLength, fm.freeSpaces[0].start, fm.freeSpaces[0].length)
	}

	start, _, err := fm.getFreeRange(52)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 15 {
		t.Errorf("Expected start to be 15, got %d", start)
	}
}
