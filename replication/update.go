package replication

import (
	"bytes"

	pb "github.com/smhanov/syzgydb/replication/proto"
)

type UpdateType int32

const (
	DeleteRecord   UpdateType = 0
	UpsertRecord   UpdateType = 1
	CreateDatabase UpdateType = 2
	DropDatabase   UpdateType = 3
)

type Update struct {
    Timestamp    Timestamp   `json:"timestamp"`
    Type         UpdateType  `json:"type"`
    Data         []byte      `json:"data"`
    DatabaseName string      `json:"database_name"`
    Dependencies []string    `json:"dependencies"`
}

func (u Update) Compare(other Update) int {
	tsComp := u.Timestamp.Compare(other.Timestamp)
	if tsComp != 0 {
		return tsComp
	}
	return bytes.Compare(u.Data, other.Data)
}

func (u Update) toProto() *pb.Update {
	return &pb.Update{
		Timestamp:    u.Timestamp.toProto(),
		Type:         pb.Update_UpdateType(u.Type),
		Data:         u.Data,
		DatabaseName: u.DatabaseName,
		Dependencies: u.Dependencies,
	}
}

func fromProtoUpdate(pu *pb.Update) Update {
	return Update{
		Timestamp:    fromProtoTimestamp(pu.Timestamp),
		Type:         UpdateType(pu.Type),
		Data:         pu.Data,
		DatabaseName: pu.DatabaseName,
		Dependencies: pu.Dependencies,
	}
}
