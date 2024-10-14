// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v3.21.12
// source: replication/messages.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Update_UpdateType int32

const (
	Update_DELETE_RECORD   Update_UpdateType = 0
	Update_UPSERT_RECORD   Update_UpdateType = 1
	Update_CREATE_DATABASE Update_UpdateType = 2
	Update_DROP_DATABASE   Update_UpdateType = 3
)

// Enum value maps for Update_UpdateType.
var (
	Update_UpdateType_name = map[int32]string{
		0: "DELETE_RECORD",
		1: "UPSERT_RECORD",
		2: "CREATE_DATABASE",
		3: "DROP_DATABASE",
	}
	Update_UpdateType_value = map[string]int32{
		"DELETE_RECORD":   0,
		"UPSERT_RECORD":   1,
		"CREATE_DATABASE": 2,
		"DROP_DATABASE":   3,
	}
)

func (x Update_UpdateType) Enum() *Update_UpdateType {
	p := new(Update_UpdateType)
	*p = x
	return p
}

func (x Update_UpdateType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Update_UpdateType) Descriptor() protoreflect.EnumDescriptor {
	return file_replication_messages_proto_enumTypes[0].Descriptor()
}

func (Update_UpdateType) Type() protoreflect.EnumType {
	return &file_replication_messages_proto_enumTypes[0]
}

func (x Update_UpdateType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Update_UpdateType.Descriptor instead.
func (Update_UpdateType) EnumDescriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{3, 0}
}

type Message_MessageType int32

const (
	Message_GOSSIP         Message_MessageType = 0
	Message_UPDATE         Message_MessageType = 1
	Message_UPDATE_REQUEST Message_MessageType = 2
	Message_BATCH_UPDATE   Message_MessageType = 3
)

// Enum value maps for Message_MessageType.
var (
	Message_MessageType_name = map[int32]string{
		0: "GOSSIP",
		1: "UPDATE",
		2: "UPDATE_REQUEST",
		3: "BATCH_UPDATE",
	}
	Message_MessageType_value = map[string]int32{
		"GOSSIP":         0,
		"UPDATE":         1,
		"UPDATE_REQUEST": 2,
		"BATCH_UPDATE":   3,
	}
)

func (x Message_MessageType) Enum() *Message_MessageType {
	p := new(Message_MessageType)
	*p = x
	return p
}

func (x Message_MessageType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Message_MessageType) Descriptor() protoreflect.EnumDescriptor {
	return file_replication_messages_proto_enumTypes[1].Descriptor()
}

func (Message_MessageType) Type() protoreflect.EnumType {
	return &file_replication_messages_proto_enumTypes[1]
}

func (x Message_MessageType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Message_MessageType.Descriptor instead.
func (Message_MessageType) EnumDescriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{7, 0}
}

type VectorClock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Clock map[uint64]*Timestamp `protobuf:"bytes,1,rep,name=clock,proto3" json:"clock,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *VectorClock) Reset() {
	*x = VectorClock{}
	mi := &file_replication_messages_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *VectorClock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VectorClock) ProtoMessage() {}

func (x *VectorClock) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VectorClock.ProtoReflect.Descriptor instead.
func (*VectorClock) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{0}
}

func (x *VectorClock) GetClock() map[uint64]*Timestamp {
	if x != nil {
		return x.Clock
	}
	return nil
}

type Timestamp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UnixTime     int64 `protobuf:"varint,1,opt,name=unix_time,json=unixTime,proto3" json:"unix_time,omitempty"`
	LamportClock int64 `protobuf:"varint,2,opt,name=lamport_clock,json=lamportClock,proto3" json:"lamport_clock,omitempty"`
}

func (x *Timestamp) Reset() {
	*x = Timestamp{}
	mi := &file_replication_messages_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Timestamp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Timestamp) ProtoMessage() {}

func (x *Timestamp) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Timestamp.ProtoReflect.Descriptor instead.
func (*Timestamp) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{1}
}

func (x *Timestamp) GetUnixTime() int64 {
	if x != nil {
		return x.UnixTime
	}
	return 0
}

func (x *Timestamp) GetLamportClock() int64 {
	if x != nil {
		return x.LamportClock
	}
	return 0
}

type DataStream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamId uint32 `protobuf:"varint,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	Data     []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *DataStream) Reset() {
	*x = DataStream{}
	mi := &file_replication_messages_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DataStream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataStream) ProtoMessage() {}

func (x *DataStream) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataStream.ProtoReflect.Descriptor instead.
func (*DataStream) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{2}
}

func (x *DataStream) GetStreamId() uint32 {
	if x != nil {
		return x.StreamId
	}
	return 0
}

func (x *DataStream) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type Update struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId       uint64            `protobuf:"varint,6,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	Timestamp    *Timestamp        `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Type         Update_UpdateType `protobuf:"varint,2,opt,name=type,proto3,enum=proto.Update_UpdateType" json:"type,omitempty"`
	RecordId     string            `protobuf:"bytes,3,opt,name=record_id,json=recordId,proto3" json:"record_id,omitempty"`
	DataStreams  []*DataStream     `protobuf:"bytes,4,rep,name=data_streams,json=dataStreams,proto3" json:"data_streams,omitempty"`
	DatabaseName string            `protobuf:"bytes,5,opt,name=database_name,json=databaseName,proto3" json:"database_name,omitempty"`
}

func (x *Update) Reset() {
	*x = Update{}
	mi := &file_replication_messages_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Update) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Update) ProtoMessage() {}

func (x *Update) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Update.ProtoReflect.Descriptor instead.
func (*Update) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{3}
}

func (x *Update) GetNodeId() uint64 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *Update) GetTimestamp() *Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Update) GetType() Update_UpdateType {
	if x != nil {
		return x.Type
	}
	return Update_DELETE_RECORD
}

func (x *Update) GetRecordId() string {
	if x != nil {
		return x.RecordId
	}
	return ""
}

func (x *Update) GetDataStreams() []*DataStream {
	if x != nil {
		return x.DataStreams
	}
	return nil
}

func (x *Update) GetDatabaseName() string {
	if x != nil {
		return x.DatabaseName
	}
	return ""
}

type GossipMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId          string       `protobuf:"bytes,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	KnownPeers      []string     `protobuf:"bytes,2,rep,name=known_peers,json=knownPeers,proto3" json:"known_peers,omitempty"`
	LastVectorClock *VectorClock `protobuf:"bytes,3,opt,name=last_vector_clock,json=lastVectorClock,proto3" json:"last_vector_clock,omitempty"`
}

func (x *GossipMessage) Reset() {
	*x = GossipMessage{}
	mi := &file_replication_messages_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GossipMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GossipMessage) ProtoMessage() {}

func (x *GossipMessage) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GossipMessage.ProtoReflect.Descriptor instead.
func (*GossipMessage) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{4}
}

func (x *GossipMessage) GetNodeId() string {
	if x != nil {
		return x.NodeId
	}
	return ""
}

func (x *GossipMessage) GetKnownPeers() []string {
	if x != nil {
		return x.KnownPeers
	}
	return nil
}

func (x *GossipMessage) GetLastVectorClock() *VectorClock {
	if x != nil {
		return x.LastVectorClock
	}
	return nil
}

type UpdateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Since      *VectorClock `protobuf:"bytes,1,opt,name=since,proto3" json:"since,omitempty"`
	MaxResults int32        `protobuf:"varint,2,opt,name=max_results,json=maxResults,proto3" json:"max_results,omitempty"`
}

func (x *UpdateRequest) Reset() {
	*x = UpdateRequest{}
	mi := &file_replication_messages_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateRequest) ProtoMessage() {}

func (x *UpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateRequest.ProtoReflect.Descriptor instead.
func (*UpdateRequest) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{5}
}

func (x *UpdateRequest) GetSince() *VectorClock {
	if x != nil {
		return x.Since
	}
	return nil
}

func (x *UpdateRequest) GetMaxResults() int32 {
	if x != nil {
		return x.MaxResults
	}
	return 0
}

type BatchUpdate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Updates []*Update `protobuf:"bytes,1,rep,name=updates,proto3" json:"updates,omitempty"`
	HasMore bool      `protobuf:"varint,2,opt,name=has_more,json=hasMore,proto3" json:"has_more,omitempty"`
}

func (x *BatchUpdate) Reset() {
	*x = BatchUpdate{}
	mi := &file_replication_messages_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BatchUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BatchUpdate) ProtoMessage() {}

func (x *BatchUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BatchUpdate.ProtoReflect.Descriptor instead.
func (*BatchUpdate) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{6}
}

func (x *BatchUpdate) GetUpdates() []*Update {
	if x != nil {
		return x.Updates
	}
	return nil
}

func (x *BatchUpdate) GetHasMore() bool {
	if x != nil {
		return x.HasMore
	}
	return false
}

type Message struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type        Message_MessageType `protobuf:"varint,1,opt,name=type,proto3,enum=proto.Message_MessageType" json:"type,omitempty"`
	VectorClock *VectorClock        `protobuf:"bytes,6,opt,name=vector_clock,json=vectorClock,proto3" json:"vector_clock,omitempty"`
	// Types that are assignable to Content:
	//
	//	*Message_GossipMessage
	//	*Message_Update
	//	*Message_UpdateRequest
	//	*Message_BatchUpdate
	Content isMessage_Content `protobuf_oneof:"content"`
}

func (x *Message) Reset() {
	*x = Message{}
	mi := &file_replication_messages_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Message) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Message) ProtoMessage() {}

func (x *Message) ProtoReflect() protoreflect.Message {
	mi := &file_replication_messages_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Message.ProtoReflect.Descriptor instead.
func (*Message) Descriptor() ([]byte, []int) {
	return file_replication_messages_proto_rawDescGZIP(), []int{7}
}

func (x *Message) GetType() Message_MessageType {
	if x != nil {
		return x.Type
	}
	return Message_GOSSIP
}

func (x *Message) GetVectorClock() *VectorClock {
	if x != nil {
		return x.VectorClock
	}
	return nil
}

func (m *Message) GetContent() isMessage_Content {
	if m != nil {
		return m.Content
	}
	return nil
}

func (x *Message) GetGossipMessage() *GossipMessage {
	if x, ok := x.GetContent().(*Message_GossipMessage); ok {
		return x.GossipMessage
	}
	return nil
}

func (x *Message) GetUpdate() *Update {
	if x, ok := x.GetContent().(*Message_Update); ok {
		return x.Update
	}
	return nil
}

func (x *Message) GetUpdateRequest() *UpdateRequest {
	if x, ok := x.GetContent().(*Message_UpdateRequest); ok {
		return x.UpdateRequest
	}
	return nil
}

func (x *Message) GetBatchUpdate() *BatchUpdate {
	if x, ok := x.GetContent().(*Message_BatchUpdate); ok {
		return x.BatchUpdate
	}
	return nil
}

type isMessage_Content interface {
	isMessage_Content()
}

type Message_GossipMessage struct {
	GossipMessage *GossipMessage `protobuf:"bytes,2,opt,name=gossip_message,json=gossipMessage,proto3,oneof"`
}

type Message_Update struct {
	Update *Update `protobuf:"bytes,3,opt,name=update,proto3,oneof"`
}

type Message_UpdateRequest struct {
	UpdateRequest *UpdateRequest `protobuf:"bytes,4,opt,name=update_request,json=updateRequest,proto3,oneof"`
}

type Message_BatchUpdate struct {
	BatchUpdate *BatchUpdate `protobuf:"bytes,5,opt,name=batch_update,json=batchUpdate,proto3,oneof"`
}

func (*Message_GossipMessage) isMessage_Content() {}

func (*Message_Update) isMessage_Content() {}

func (*Message_UpdateRequest) isMessage_Content() {}

func (*Message_BatchUpdate) isMessage_Content() {}

var File_replication_messages_proto protoreflect.FileDescriptor

var file_replication_messages_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x8e, 0x01, 0x0a, 0x0b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x43, 0x6c,
	0x6f, 0x63, 0x6b, 0x12, 0x33, 0x0a, 0x05, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x05, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x1a, 0x4a, 0x0a, 0x0a, 0x43, 0x6c, 0x6f, 0x63,
	0x6b, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x26, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0x4d, 0x0a, 0x09, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x12, 0x1b, 0x0a, 0x09, 0x75, 0x6e, 0x69, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x75, 0x6e, 0x69, 0x78, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x23,
	0x0a, 0x0d, 0x6c, 0x61, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x6c, 0x61, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x43, 0x6c,
	0x6f, 0x63, 0x6b, 0x22, 0x3d, 0x0a, 0x0a, 0x44, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61,
	0x74, 0x61, 0x22, 0xd3, 0x02, 0x0a, 0x06, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x17, 0x0a,
	0x07, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06,
	0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12, 0x2e, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x2c, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x5f, 0x69,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x49,
	0x64, 0x12, 0x34, 0x0a, 0x0c, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x0b, 0x64, 0x61, 0x74, 0x61,
	0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x64, 0x61, 0x74, 0x61, 0x62,
	0x61, 0x73, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x5a, 0x0a, 0x0a,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x11, 0x0a, 0x0d, 0x44, 0x45,
	0x4c, 0x45, 0x54, 0x45, 0x5f, 0x52, 0x45, 0x43, 0x4f, 0x52, 0x44, 0x10, 0x00, 0x12, 0x11, 0x0a,
	0x0d, 0x55, 0x50, 0x53, 0x45, 0x52, 0x54, 0x5f, 0x52, 0x45, 0x43, 0x4f, 0x52, 0x44, 0x10, 0x01,
	0x12, 0x13, 0x0a, 0x0f, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x5f, 0x44, 0x41, 0x54, 0x41, 0x42,
	0x41, 0x53, 0x45, 0x10, 0x02, 0x12, 0x11, 0x0a, 0x0d, 0x44, 0x52, 0x4f, 0x50, 0x5f, 0x44, 0x41,
	0x54, 0x41, 0x42, 0x41, 0x53, 0x45, 0x10, 0x03, 0x22, 0x89, 0x01, 0x0a, 0x0d, 0x47, 0x6f, 0x73,
	0x73, 0x69, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x6e, 0x6f,
	0x64, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6e, 0x6f, 0x64,
	0x65, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x5f, 0x70, 0x65, 0x65,
	0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x50,
	0x65, 0x65, 0x72, 0x73, 0x12, 0x3e, 0x0a, 0x11, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x76, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x5f, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x43, 0x6c,
	0x6f, 0x63, 0x6b, 0x52, 0x0f, 0x6c, 0x61, 0x73, 0x74, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x43,
	0x6c, 0x6f, 0x63, 0x6b, 0x22, 0x5a, 0x0a, 0x0d, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x28, 0x0a, 0x05, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x56, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x05, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x12,
	0x1f, 0x0a, 0x0b, 0x6d, 0x61, 0x78, 0x5f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x6d, 0x61, 0x78, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73,
	0x22, 0x51, 0x0a, 0x0b, 0x42, 0x61, 0x74, 0x63, 0x68, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12,
	0x27, 0x0a, 0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52,
	0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x73, 0x12, 0x19, 0x0a, 0x08, 0x68, 0x61, 0x73, 0x5f,
	0x6d, 0x6f, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x68, 0x61, 0x73, 0x4d,
	0x6f, 0x72, 0x65, 0x22, 0xa8, 0x03, 0x0a, 0x07, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x2e, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x35, 0x0a, 0x0c, 0x76, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x5f, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x56, 0x65,
	0x63, 0x74, 0x6f, 0x72, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x0b, 0x76, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x3d, 0x0a, 0x0e, 0x67, 0x6f, 0x73, 0x73, 0x69, 0x70,
	0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x6f, 0x73, 0x73, 0x69, 0x70, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x48, 0x00, 0x52, 0x0d, 0x67, 0x6f, 0x73, 0x73, 0x69, 0x70, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x27, 0x0a, 0x06, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x55, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x48, 0x00, 0x52, 0x06, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x3d,
	0x0a, 0x0e, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x55,
	0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0d,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x37, 0x0a,
	0x0c, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x61, 0x74, 0x63,
	0x68, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x48, 0x00, 0x52, 0x0b, 0x62, 0x61, 0x74, 0x63, 0x68,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x22, 0x4b, 0x0a, 0x0b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x47, 0x4f, 0x53, 0x53, 0x49, 0x50, 0x10,
	0x00, 0x12, 0x0a, 0x0a, 0x06, 0x55, 0x50, 0x44, 0x41, 0x54, 0x45, 0x10, 0x01, 0x12, 0x12, 0x0a,
	0x0e, 0x55, 0x50, 0x44, 0x41, 0x54, 0x45, 0x5f, 0x52, 0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x10,
	0x02, 0x12, 0x10, 0x0a, 0x0c, 0x42, 0x41, 0x54, 0x43, 0x48, 0x5f, 0x55, 0x50, 0x44, 0x41, 0x54,
	0x45, 0x10, 0x03, 0x42, 0x09, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x42, 0x13,
	0x5a, 0x11, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_replication_messages_proto_rawDescOnce sync.Once
	file_replication_messages_proto_rawDescData = file_replication_messages_proto_rawDesc
)

func file_replication_messages_proto_rawDescGZIP() []byte {
	file_replication_messages_proto_rawDescOnce.Do(func() {
		file_replication_messages_proto_rawDescData = protoimpl.X.CompressGZIP(file_replication_messages_proto_rawDescData)
	})
	return file_replication_messages_proto_rawDescData
}

var file_replication_messages_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_replication_messages_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_replication_messages_proto_goTypes = []any{
	(Update_UpdateType)(0),   // 0: proto.Update.UpdateType
	(Message_MessageType)(0), // 1: proto.Message.MessageType
	(*VectorClock)(nil),      // 2: proto.VectorClock
	(*Timestamp)(nil),        // 3: proto.Timestamp
	(*DataStream)(nil),       // 4: proto.DataStream
	(*Update)(nil),           // 5: proto.Update
	(*GossipMessage)(nil),    // 6: proto.GossipMessage
	(*UpdateRequest)(nil),    // 7: proto.UpdateRequest
	(*BatchUpdate)(nil),      // 8: proto.BatchUpdate
	(*Message)(nil),          // 9: proto.Message
	nil,                      // 10: proto.VectorClock.ClockEntry
}
var file_replication_messages_proto_depIdxs = []int32{
	10, // 0: proto.VectorClock.clock:type_name -> proto.VectorClock.ClockEntry
	3,  // 1: proto.Update.timestamp:type_name -> proto.Timestamp
	0,  // 2: proto.Update.type:type_name -> proto.Update.UpdateType
	4,  // 3: proto.Update.data_streams:type_name -> proto.DataStream
	2,  // 4: proto.GossipMessage.last_vector_clock:type_name -> proto.VectorClock
	2,  // 5: proto.UpdateRequest.since:type_name -> proto.VectorClock
	5,  // 6: proto.BatchUpdate.updates:type_name -> proto.Update
	1,  // 7: proto.Message.type:type_name -> proto.Message.MessageType
	2,  // 8: proto.Message.vector_clock:type_name -> proto.VectorClock
	6,  // 9: proto.Message.gossip_message:type_name -> proto.GossipMessage
	5,  // 10: proto.Message.update:type_name -> proto.Update
	7,  // 11: proto.Message.update_request:type_name -> proto.UpdateRequest
	8,  // 12: proto.Message.batch_update:type_name -> proto.BatchUpdate
	3,  // 13: proto.VectorClock.ClockEntry.value:type_name -> proto.Timestamp
	14, // [14:14] is the sub-list for method output_type
	14, // [14:14] is the sub-list for method input_type
	14, // [14:14] is the sub-list for extension type_name
	14, // [14:14] is the sub-list for extension extendee
	0,  // [0:14] is the sub-list for field type_name
}

func init() { file_replication_messages_proto_init() }
func file_replication_messages_proto_init() {
	if File_replication_messages_proto != nil {
		return
	}
	file_replication_messages_proto_msgTypes[7].OneofWrappers = []any{
		(*Message_GossipMessage)(nil),
		(*Message_Update)(nil),
		(*Message_UpdateRequest)(nil),
		(*Message_BatchUpdate)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_replication_messages_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_replication_messages_proto_goTypes,
		DependencyIndexes: file_replication_messages_proto_depIdxs,
		EnumInfos:         file_replication_messages_proto_enumTypes,
		MessageInfos:      file_replication_messages_proto_msgTypes,
	}.Build()
	File_replication_messages_proto = out.File
	file_replication_messages_proto_rawDesc = nil
	file_replication_messages_proto_goTypes = nil
	file_replication_messages_proto_depIdxs = nil
}
