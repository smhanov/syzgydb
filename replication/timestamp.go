package replication

import (
	"bytes"
	"fmt"
	"time"

	pb "github.com/smhanov/syzgydb/replication/proto"
)

// Timestamp represents a hybrid logical clock timestamp used in the replication system.
type Timestamp struct {
	UnixTime     int64 `json:"unix_time"`
	LamportClock int64 `json:"lamport_clock"`
}

// Now creates a new Timestamp representing the current time.
func Now() Timestamp {
	return Timestamp{
		UnixTime:     time.Now().UnixNano() / int64(time.Millisecond),
		LamportClock: 0, // Initialize LamportClock to 0 or another appropriate value
	}
}

// Compare compares two Timestamps and returns:
//
//	-1 if t < other
//	 0 if t == other
//	 1 if t > other
func (t Timestamp) Compare(other Timestamp) int {
	if t.LamportClock < other.LamportClock {
		return -1
	} else if t.LamportClock > other.LamportClock {
		return 1
	} else {
		if t.UnixTime < other.UnixTime {
			return -1
		} else if t.UnixTime > other.UnixTime {
			return 1
		}
	}
	return 0
}

// Next returns a new Timestamp with the LamportClock incremented by 1.
func (t Timestamp) Next(local bool) Timestamp {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	/*if local {
		return Timestamp{
			UnixTime:     now,
			LamportClock: t.LamportClock,
		}
	}*/
	return Timestamp{
		UnixTime:     now,
		LamportClock: t.LamportClock + 1,
	}
}

// IsLessThan returns true if the current Timestamp is less than the given Timestamp, false otherwise.
func (t Timestamp) After(other Timestamp) bool {
	return t.Compare(other) > 0
}

// Before returns true if the current Timestamp is before the given Timestamp, false otherwise.
func (t Timestamp) Before(other Timestamp) bool {
	return t.Compare(other) < 0
}

// Equal checks if two Timestamps are equal.
func (t Timestamp) Equal(other Timestamp) bool {
	return t.LamportClock == other.LamportClock && t.UnixTime == other.UnixTime
}

// Bytes serializes the Timestamp into a byte slice.
func (t Timestamp) Bytes() []byte {
	buf := new(bytes.Buffer)
	// Implement serialization logic here
	return buf.Bytes()
}

// toProto converts a Timestamp to its protobuf representation.
func (t Timestamp) toProto() *pb.Timestamp {
	return &pb.Timestamp{
		UnixTime:     t.UnixTime,
		LamportClock: t.LamportClock,
	}
}

// fromProtoTimestamp converts a protobuf Timestamp to a Timestamp struct.
func fromProtoTimestamp(pt *pb.Timestamp) Timestamp {
	return Timestamp{
		UnixTime:     pt.UnixTime,
		LamportClock: pt.LamportClock,
	}
}

// String returns a string representation of the Timestamp.
// The format is the lamport clock in hex followed by [YYYY-MM-DD/hh:mm:ss.ms]
func (t Timestamp) String() string {
	unixTime := time.Unix(0, t.UnixTime*int64(time.Millisecond))
	return fmt.Sprintf("%x[%s]",
		t.LamportClock,
		unixTime.Format("2006-01-02/15:04:05.000"))
}

func (t Timestamp) IsZero() bool {
	return t.LamportClock == 0 && t.UnixTime == 0
}
