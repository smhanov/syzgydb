package replication

import (
	"encoding/json"
	"sync"

	"github.com/smhanov/syzgydb/replication/proto"
)

// NodeSequences is a thread-safe map from uint64 nodeID to uint64 sequenceNumbers
type NodeSequences struct {
	sequences map[uint64]uint64
	mutex     sync.RWMutex
}

// NewNodeSequences creates a new NodeSequences instance
func NewNodeSequences() *NodeSequences {
	return &NodeSequences{
		sequences: make(map[uint64]uint64),
	}
}

// Get returns the sequence number for a given nodeID
func (ns *NodeSequences) Get(nodeID uint64) (uint64, bool) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	seq, ok := ns.sequences[nodeID]
	return seq, ok
}

// Before returns true if the node id doesn't exist or if the sequence number is before the recorded one for the node
func (ns *NodeSequences) Before(nodeID uint64, sequenceNumber uint64) bool {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	currentSeq, exists := ns.sequences[nodeID]
	return !exists || sequenceNumber < currentSeq
}

// Update updates the sequence number of a particular node
func (ns *NodeSequences) Update(nodeID uint64, sequenceNumber uint64) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	ns.sequences[nodeID] = sequenceNumber
}

// MarshalJSON implements the json.Marshaler interface
func (ns *NodeSequences) MarshalJSON() ([]byte, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	return json.Marshal(ns.sequences)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (ns *NodeSequences) UnmarshalJSON(data []byte) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	return json.Unmarshal(data, &ns.sequences)
}

// Clone creates a deep copy of the NodeSequences
func (ns *NodeSequences) Clone() *NodeSequences {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	clone := NewNodeSequences()
	for nodeID, seq := range ns.sequences {
		clone.sequences[nodeID] = seq
	}
	return clone
}

// ToProto converts the NodeSequences to its protobuf representation
func (ns *NodeSequences) toProto() *proto.NodeSequences {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	protoNS := &proto.NodeSequences{
		Clock: make(map[uint64]uint64, len(ns.sequences)),
	}

	for nodeID, seq := range ns.sequences {
		protoNS.Clock[nodeID] = seq
	}

	return protoNS
}

// fromProtoNodeSequences converts a protobuf NodeSequences to a NodeSequences
func fromProtoNodeSequences(protoNS *proto.NodeSequences) *NodeSequences {
	ns := NewNodeSequences()
	for nodeID, seq := range protoNS.Clock {
		ns.Update(nodeID, seq)
	}
	return ns
}
