package syzygy

import (
	"testing"
)

func TestMarkFreeAndGetFreeRange(t *testing.T) {
	fm := &FreeMap{}

	// Mark some spaces as free
	fm.markFree(0, 10)
	fm.markFree(20, 5)
	fm.markFree(15, 5)

	// Test merging of contiguous spaces
	if len(fm.freeSpaces) != 2 {
		t.Errorf("Expected 2 free spaces, got %d", len(fm.freeSpaces))
	}

	// Test getting a free range
	start, err := fm.getFreeRange(5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 0 {
		t.Errorf("Expected start to be 0, got %d", start)
	}

	// Test getting another free range
	start, err = fm.getFreeRange(10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if start != 15 {
		t.Errorf("Expected start to be 15, got %d", start)
	}

	// Test insufficient space
	_, err = fm.getFreeRange(10)
	if err == nil {
		t.Errorf("Expected error due to insufficient space, got nil")
	}
}
