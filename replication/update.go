package replication

import (
	"bytes"

	pb "github.com/smhanov/syzgydb/replication/proto"
)

// UpdateType represents the type of update operation.
type UpdateType int32

const (
	DeleteRecord   UpdateType = 0
	UpsertRecord   UpdateType = 1
	CreateDatabase UpdateType = 2
	DropDatabase   UpdateType = 3
)

type DataStream struct {
    StreamID uint8
    Data     []byte
}

// Update represents a single update operation in the replication system.
type Update struct {
    Timestamp    Timestamp   `json:"timestamp"`
    Type         UpdateType  `json:"type"`
    RecordID     string      `json:"record_id"`
    DataStreams  []DataStream `json:"data_streams"`
    DatabaseName string      `json:"database_name"`
    Dependencies []string    `json:"dependencies"`
}

// Compare compares two Updates based on their timestamps and record IDs.
func (u Update) Compare(other Update) int {
    tsComp := u.Timestamp.Compare(other.Timestamp)
    if tsComp != 0 {
        return tsComp
    }
    return bytes.Compare([]byte(u.RecordID), []byte(other.RecordID))
}

// toProto converts an Update to its protobuf representation.
func (u Update) toProto() *pb.Update {
    protoDataStreams := make([]*pb.DataStream, len(u.DataStreams))
    for i, ds := range u.DataStreams {
        protoDataStreams[i] = &pb.DataStream{
            StreamId: uint32(ds.StreamID),
            Data:     ds.Data,
        }
    }
    return &pb.Update{
        Timestamp:    u.Timestamp.toProto(),
        Type:         pb.Update_UpdateType(u.Type),
        RecordId:     u.RecordID,
        DataStreams:  protoDataStreams,
        DatabaseName: u.DatabaseName,
        Dependencies: u.Dependencies,
    }
}

// fromProtoUpdate converts a protobuf Update to an Update struct.
func fromProtoUpdate(pu *pb.Update) Update {
    dataStreams := make([]DataStream, len(pu.DataStreams))
    for i, pds := range pu.DataStreams {
        dataStreams[i] = DataStream{
            StreamID: uint8(pds.StreamId),
            Data:     pds.Data,
        }
    }
    return Update{
        Timestamp:    fromProtoTimestamp(pu.Timestamp),
        Type:         UpdateType(pu.Type),
        RecordID:     pu.RecordId,
        DataStreams:  dataStreams,
        DatabaseName: pu.DatabaseName,
        Dependencies: pu.Dependencies,
    }
}
