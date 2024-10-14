package replication

import (
	"fmt"
	"sort"
)

// VectorClock represents a vector clock for distributed systems.
type VectorClock struct {
	clock map[uint64]Timestamp
}

// NewVectorClock creates a new, empty VectorClock.
func NewVectorClock() *VectorClock {
	return &VectorClock{
		clock: make(map[uint64]Timestamp),
	}
}

// Update updates the vector clock for a given nodeID with a new Timestamp.
func (vc *VectorClock) Update(nodeID uint64, timestamp Timestamp) {
	vc.clock[nodeID] = timestamp
}

// Get retrieves the Timestamp for a given nodeID.
func (vc *VectorClock) Get(nodeID uint64) (Timestamp, bool) {
	ts, ok := vc.clock[nodeID]
	return ts, ok
}

// Contains checks if the vector clock contains a particular nodeID.
func (vc *VectorClock) Contains(nodeID uint64) bool {
	_, ok := vc.clock[nodeID]
	return ok
}

// Before checks if the current VectorClock is causally before another VectorClock.
func (vc *VectorClock) Before(other *VectorClock) bool {
	if len(vc.clock) > len(other.clock) {
		return false
	}

	for nodeID, ts := range vc.clock {
		otherTS, ok := other.clock[nodeID]
		if !ok || ts.After(otherTS) {
			return false
		}
	}

	return len(vc.clock) < len(other.clock) || !vc.Equal(other)
}

// After checks if the current VectorClock is causally after another VectorClock.
func (vc *VectorClock) After(other *VectorClock) bool {
	return other.Before(vc)
}

// Equal checks if two VectorClocks are equal.
func (vc *VectorClock) Equal(other *VectorClock) bool {
	if len(vc.clock) != len(other.clock) {
		return false
	}

	for nodeID, ts := range vc.clock {
		otherTS, ok := other.clock[nodeID]
		if !ok || !ts.Equal(otherTS) {
			return false
		}
	}

	return true
}

// Merge merges another VectorClock into the current one, taking the maximum Timestamp for each nodeID.
func (vc *VectorClock) Merge(other *VectorClock) {
	for nodeID, otherTS := range other.clock {
		if currentTS, ok := vc.clock[nodeID]; !ok || otherTS.After(currentTS) {
			vc.clock[nodeID] = otherTS
		}
	}
}

// String returns a string representation of the VectorClock.
func (vc *VectorClock) String() string {
	var nodeIDs []uint64
	for nodeID := range vc.clock {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })

	result := "{"
	for i, nodeID := range nodeIDs {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d: %s", nodeID, vc.clock[nodeID].String())
	}
	result += "}"
	return result
}
