package replication

import (
	pb "github.com/smhanov/syzgydb/replication/proto"
)

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
		NodeId:       u.NodeID,
		Timestamp:    u.Timestamp.toProto(),
		Type:         pb.Update_UpdateType(u.Type),
		RecordId:     u.RecordID,
		DataStreams:  protoDataStreams,
		DatabaseName: u.DatabaseName,
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
		NodeID:       pu.NodeId,
		Timestamp:    fromProtoTimestamp(pu.Timestamp),
		Type:         UpdateType(pu.Type),
		RecordID:     pu.RecordId,
		DataStreams:  dataStreams,
		DatabaseName: pu.DatabaseName,
	}
}
