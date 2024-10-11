package replication

import "bytes"

type Timestamp struct {
    UnixTime     int64 `json:"unix_time"`
    LamportClock int64 `json:"lamport_clock"`
}

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

func (t Timestamp) Equal(other Timestamp) bool {
    return t.LamportClock == other.LamportClock && t.UnixTime == other.UnixTime
}

func (t Timestamp) Bytes() []byte {
    buf := new(bytes.Buffer)
    // Implement serialization logic here
    return buf.Bytes()
}

func (t Timestamp) toProto() *pb.Timestamp {
    return &pb.Timestamp{
        UnixTime:     t.UnixTime,
        LamportClock: t.LamportClock,
    }
}

func fromProtoTimestamp(pt *pb.Timestamp) Timestamp {
    return Timestamp{
        UnixTime:     pt.UnixTime,
        LamportClock: pt.LamportClock,
    }
}
