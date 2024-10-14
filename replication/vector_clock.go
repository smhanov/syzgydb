package replication

import (
	"fmt"
	"sort"

	pb "github.com/smhanov/syzgydb/replication/proto"
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
func (vc *VectorClock) Update(nodeID uint64, timestamp Timestamp) *VectorClock {
	if ts, exists := vc.clock[nodeID]; exists && ts.Before(timestamp) || !exists {
		vc.clock[nodeID] = timestamp
	}
	return vc
}

// Get retrieves the Timestamp for a given nodeID.
func (vc *VectorClock) Get(nodeID uint64) (Timestamp, bool) {
	ts, ok := vc.clock[nodeID]
	return ts, ok
}

func (vc *VectorClock) BeforeTimestamp(nodeID uint64, ts Timestamp) bool {
	if currentTS, ok := vc.clock[nodeID]; ok {
		return currentTS.Before(ts)
	}

	return true
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

// Compare compares two VectorClocks and returns:
//
//	-1 if vc < other
//	 0 if vc == other
//	 1 if vc > other
func (vc *VectorClock) Compare(other *VectorClock) int {
	if vc.Equal(other) {
		return 0
	}
	if vc.Before(other) {
		return -1
	}
	return 1
}

// Clone returns a deep copy of the VectorClock.
func (vc *VectorClock) Clone() *VectorClock {
	clone := NewVectorClock()
	for nodeID, ts := range vc.clock {
		clone.clock[nodeID] = ts
	}
	return clone
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

func (vc *VectorClock) toProto() *pb.VectorClock {
	protoVC := &pb.VectorClock{
		Clock: make(map[uint64]*pb.Timestamp),
	}
	for nodeID, ts := range vc.clock {
		protoVC.Clock[nodeID] = ts.toProto()
	}
	return protoVC
}

func fromProtoVectorClock(pvc *pb.VectorClock) *VectorClock {
	vc := NewVectorClock()
	for nodeID, pts := range pvc.Clock {
		vc.clock[nodeID] = fromProtoTimestamp(pts)
	}
	return vc
}
