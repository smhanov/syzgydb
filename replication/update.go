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

// Update represents a single update operation in the replication system.
type Update struct {
    Timestamp    Timestamp   `json:"timestamp"`
    Type         UpdateType  `json:"type"`
    Data         []byte      `json:"data"`
    DatabaseName string      `json:"database_name"`
    Dependencies []string    `json:"dependencies"`
}

// Compare compares two Updates based on their timestamps and data.
func (u Update) Compare(other Update) int {
	tsComp := u.Timestamp.Compare(other.Timestamp)
	if tsComp != 0 {
		return tsComp
	}
	return bytes.Compare(u.Data, other.Data)
}

// toProto converts an Update to its protobuf representation.
func (u Update) toProto() *pb.Update {
	return &pb.Update{
		Timestamp:    u.Timestamp.toProto(),
		Type:         pb.Update_UpdateType(u.Type),
		Data:         u.Data,
		DatabaseName: u.DatabaseName,
		Dependencies: u.Dependencies,
	}
}

// fromProtoUpdate converts a protobuf Update to an Update struct.
func fromProtoUpdate(pu *pb.Update) Update {
	return Update{
		Timestamp:    fromProtoTimestamp(pu.Timestamp),
		Type:         UpdateType(pu.Type),
		Data:         pu.Data,
		DatabaseName: pu.DatabaseName,
		Dependencies: pu.Dependencies,
	}
}
