// Code generated by protoc-gen-go. DO NOT EDIT.
// source: language-agent-v2/trace.proto

package language_agent_v2

import (
	context "context"
	fmt "fmt"
	common "github.com/SkyAPM/go2sky/reporter/grpc/common"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type SegmentObject struct {
	TraceSegmentId       *common.UniqueId `protobuf:"bytes,1,opt,name=traceSegmentId,proto3" json:"traceSegmentId,omitempty"`
	Spans                []*SpanObjectV2  `protobuf:"bytes,2,rep,name=spans,proto3" json:"spans,omitempty"`
	ServiceId            int32            `protobuf:"varint,3,opt,name=serviceId,proto3" json:"serviceId,omitempty"`
	ServiceInstanceId    int32            `protobuf:"varint,4,opt,name=serviceInstanceId,proto3" json:"serviceInstanceId,omitempty"`
	IsSizeLimited        bool             `protobuf:"varint,5,opt,name=isSizeLimited,proto3" json:"isSizeLimited,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *SegmentObject) Reset()         { *m = SegmentObject{} }
func (m *SegmentObject) String() string { return proto.CompactTextString(m) }
func (*SegmentObject) ProtoMessage()    {}
func (*SegmentObject) Descriptor() ([]byte, []int) {
	return fileDescriptor_8124ab206744863a, []int{0}
}

func (m *SegmentObject) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SegmentObject.Unmarshal(m, b)
}
func (m *SegmentObject) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SegmentObject.Marshal(b, m, deterministic)
}
func (m *SegmentObject) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SegmentObject.Merge(m, src)
}
func (m *SegmentObject) XXX_Size() int {
	return xxx_messageInfo_SegmentObject.Size(m)
}
func (m *SegmentObject) XXX_DiscardUnknown() {
	xxx_messageInfo_SegmentObject.DiscardUnknown(m)
}

var xxx_messageInfo_SegmentObject proto.InternalMessageInfo

func (m *SegmentObject) GetTraceSegmentId() *common.UniqueId {
	if m != nil {
		return m.TraceSegmentId
	}
	return nil
}

func (m *SegmentObject) GetSpans() []*SpanObjectV2 {
	if m != nil {
		return m.Spans
	}
	return nil
}

func (m *SegmentObject) GetServiceId() int32 {
	if m != nil {
		return m.ServiceId
	}
	return 0
}

func (m *SegmentObject) GetServiceInstanceId() int32 {
	if m != nil {
		return m.ServiceInstanceId
	}
	return 0
}

func (m *SegmentObject) GetIsSizeLimited() bool {
	if m != nil {
		return m.IsSizeLimited
	}
	return false
}

type SegmentReference struct {
	RefType                 common.RefType   `protobuf:"varint,1,opt,name=refType,proto3,enum=RefType" json:"refType,omitempty"`
	ParentTraceSegmentId    *common.UniqueId `protobuf:"bytes,2,opt,name=parentTraceSegmentId,proto3" json:"parentTraceSegmentId,omitempty"`
	ParentSpanId            int32            `protobuf:"varint,3,opt,name=parentSpanId,proto3" json:"parentSpanId,omitempty"`
	ParentServiceInstanceId int32            `protobuf:"varint,4,opt,name=parentServiceInstanceId,proto3" json:"parentServiceInstanceId,omitempty"`
	NetworkAddress          string           `protobuf:"bytes,5,opt,name=networkAddress,proto3" json:"networkAddress,omitempty"`
	NetworkAddressId        int32            `protobuf:"varint,6,opt,name=networkAddressId,proto3" json:"networkAddressId,omitempty"`
	EntryServiceInstanceId  int32            `protobuf:"varint,7,opt,name=entryServiceInstanceId,proto3" json:"entryServiceInstanceId,omitempty"`
	EntryEndpoint           string           `protobuf:"bytes,8,opt,name=entryEndpoint,proto3" json:"entryEndpoint,omitempty"`
	EntryEndpointId         int32            `protobuf:"varint,9,opt,name=entryEndpointId,proto3" json:"entryEndpointId,omitempty"`
	ParentEndpoint          string           `protobuf:"bytes,10,opt,name=parentEndpoint,proto3" json:"parentEndpoint,omitempty"`
	ParentEndpointId        int32            `protobuf:"varint,11,opt,name=parentEndpointId,proto3" json:"parentEndpointId,omitempty"`
	XXX_NoUnkeyedLiteral    struct{}         `json:"-"`
	XXX_unrecognized        []byte           `json:"-"`
	XXX_sizecache           int32            `json:"-"`
}

func (m *SegmentReference) Reset()         { *m = SegmentReference{} }
func (m *SegmentReference) String() string { return proto.CompactTextString(m) }
func (*SegmentReference) ProtoMessage()    {}
func (*SegmentReference) Descriptor() ([]byte, []int) {
	return fileDescriptor_8124ab206744863a, []int{1}
}

func (m *SegmentReference) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SegmentReference.Unmarshal(m, b)
}
func (m *SegmentReference) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SegmentReference.Marshal(b, m, deterministic)
}
func (m *SegmentReference) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SegmentReference.Merge(m, src)
}
func (m *SegmentReference) XXX_Size() int {
	return xxx_messageInfo_SegmentReference.Size(m)
}
func (m *SegmentReference) XXX_DiscardUnknown() {
	xxx_messageInfo_SegmentReference.DiscardUnknown(m)
}

var xxx_messageInfo_SegmentReference proto.InternalMessageInfo

func (m *SegmentReference) GetRefType() common.RefType {
	if m != nil {
		return m.RefType
	}
	return common.RefType_CrossProcess
}

func (m *SegmentReference) GetParentTraceSegmentId() *common.UniqueId {
	if m != nil {
		return m.ParentTraceSegmentId
	}
	return nil
}

func (m *SegmentReference) GetParentSpanId() int32 {
	if m != nil {
		return m.ParentSpanId
	}
	return 0
}

func (m *SegmentReference) GetParentServiceInstanceId() int32 {
	if m != nil {
		return m.ParentServiceInstanceId
	}
	return 0
}

func (m *SegmentReference) GetNetworkAddress() string {
	if m != nil {
		return m.NetworkAddress
	}
	return ""
}

func (m *SegmentReference) GetNetworkAddressId() int32 {
	if m != nil {
		return m.NetworkAddressId
	}
	return 0
}

func (m *SegmentReference) GetEntryServiceInstanceId() int32 {
	if m != nil {
		return m.EntryServiceInstanceId
	}
	return 0
}

func (m *SegmentReference) GetEntryEndpoint() string {
	if m != nil {
		return m.EntryEndpoint
	}
	return ""
}

func (m *SegmentReference) GetEntryEndpointId() int32 {
	if m != nil {
		return m.EntryEndpointId
	}
	return 0
}

func (m *SegmentReference) GetParentEndpoint() string {
	if m != nil {
		return m.ParentEndpoint
	}
	return ""
}

func (m *SegmentReference) GetParentEndpointId() int32 {
	if m != nil {
		return m.ParentEndpointId
	}
	return 0
}

type SpanObjectV2 struct {
	SpanId               int32                        `protobuf:"varint,1,opt,name=spanId,proto3" json:"spanId,omitempty"`
	ParentSpanId         int32                        `protobuf:"varint,2,opt,name=parentSpanId,proto3" json:"parentSpanId,omitempty"`
	StartTime            int64                        `protobuf:"varint,3,opt,name=startTime,proto3" json:"startTime,omitempty"`
	EndTime              int64                        `protobuf:"varint,4,opt,name=endTime,proto3" json:"endTime,omitempty"`
	Refs                 []*SegmentReference          `protobuf:"bytes,5,rep,name=refs,proto3" json:"refs,omitempty"`
	OperationNameId      int32                        `protobuf:"varint,6,opt,name=operationNameId,proto3" json:"operationNameId,omitempty"`
	OperationName        string                       `protobuf:"bytes,7,opt,name=operationName,proto3" json:"operationName,omitempty"`
	PeerId               int32                        `protobuf:"varint,8,opt,name=peerId,proto3" json:"peerId,omitempty"`
	Peer                 string                       `protobuf:"bytes,9,opt,name=peer,proto3" json:"peer,omitempty"`
	SpanType             common.SpanType              `protobuf:"varint,10,opt,name=spanType,proto3,enum=SpanType" json:"spanType,omitempty"`
	SpanLayer            common.SpanLayer             `protobuf:"varint,11,opt,name=spanLayer,proto3,enum=SpanLayer" json:"spanLayer,omitempty"`
	ComponentId          int32                        `protobuf:"varint,12,opt,name=componentId,proto3" json:"componentId,omitempty"`
	Component            string                       `protobuf:"bytes,13,opt,name=component,proto3" json:"component,omitempty"`
	IsError              bool                         `protobuf:"varint,14,opt,name=isError,proto3" json:"isError,omitempty"`
	Tags                 []*common.KeyStringValuePair `protobuf:"bytes,15,rep,name=tags,proto3" json:"tags,omitempty"`
	Logs                 []*Log                       `protobuf:"bytes,16,rep,name=logs,proto3" json:"logs,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *SpanObjectV2) Reset()         { *m = SpanObjectV2{} }
func (m *SpanObjectV2) String() string { return proto.CompactTextString(m) }
func (*SpanObjectV2) ProtoMessage()    {}
func (*SpanObjectV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_8124ab206744863a, []int{2}
}

func (m *SpanObjectV2) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SpanObjectV2.Unmarshal(m, b)
}
func (m *SpanObjectV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SpanObjectV2.Marshal(b, m, deterministic)
}
func (m *SpanObjectV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SpanObjectV2.Merge(m, src)
}
func (m *SpanObjectV2) XXX_Size() int {
	return xxx_messageInfo_SpanObjectV2.Size(m)
}
func (m *SpanObjectV2) XXX_DiscardUnknown() {
	xxx_messageInfo_SpanObjectV2.DiscardUnknown(m)
}

var xxx_messageInfo_SpanObjectV2 proto.InternalMessageInfo

func (m *SpanObjectV2) GetSpanId() int32 {
	if m != nil {
		return m.SpanId
	}
	return 0
}

func (m *SpanObjectV2) GetParentSpanId() int32 {
	if m != nil {
		return m.ParentSpanId
	}
	return 0
}

func (m *SpanObjectV2) GetStartTime() int64 {
	if m != nil {
		return m.StartTime
	}
	return 0
}

func (m *SpanObjectV2) GetEndTime() int64 {
	if m != nil {
		return m.EndTime
	}
	return 0
}

func (m *SpanObjectV2) GetRefs() []*SegmentReference {
	if m != nil {
		return m.Refs
	}
	return nil
}

func (m *SpanObjectV2) GetOperationNameId() int32 {
	if m != nil {
		return m.OperationNameId
	}
	return 0
}

func (m *SpanObjectV2) GetOperationName() string {
	if m != nil {
		return m.OperationName
	}
	return ""
}

func (m *SpanObjectV2) GetPeerId() int32 {
	if m != nil {
		return m.PeerId
	}
	return 0
}

func (m *SpanObjectV2) GetPeer() string {
	if m != nil {
		return m.Peer
	}
	return ""
}

func (m *SpanObjectV2) GetSpanType() common.SpanType {
	if m != nil {
		return m.SpanType
	}
	return common.SpanType_Entry
}

func (m *SpanObjectV2) GetSpanLayer() common.SpanLayer {
	if m != nil {
		return m.SpanLayer
	}
	return common.SpanLayer_Unknown
}

func (m *SpanObjectV2) GetComponentId() int32 {
	if m != nil {
		return m.ComponentId
	}
	return 0
}

func (m *SpanObjectV2) GetComponent() string {
	if m != nil {
		return m.Component
	}
	return ""
}

func (m *SpanObjectV2) GetIsError() bool {
	if m != nil {
		return m.IsError
	}
	return false
}

func (m *SpanObjectV2) GetTags() []*common.KeyStringValuePair {
	if m != nil {
		return m.Tags
	}
	return nil
}

func (m *SpanObjectV2) GetLogs() []*Log {
	if m != nil {
		return m.Logs
	}
	return nil
}

type Log struct {
	Time                 int64                        `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Data                 []*common.KeyStringValuePair `protobuf:"bytes,2,rep,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *Log) Reset()         { *m = Log{} }
func (m *Log) String() string { return proto.CompactTextString(m) }
func (*Log) ProtoMessage()    {}
func (*Log) Descriptor() ([]byte, []int) {
	return fileDescriptor_8124ab206744863a, []int{3}
}

func (m *Log) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Log.Unmarshal(m, b)
}
func (m *Log) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Log.Marshal(b, m, deterministic)
}
func (m *Log) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Log.Merge(m, src)
}
func (m *Log) XXX_Size() int {
	return xxx_messageInfo_Log.Size(m)
}
func (m *Log) XXX_DiscardUnknown() {
	xxx_messageInfo_Log.DiscardUnknown(m)
}

var xxx_messageInfo_Log proto.InternalMessageInfo

func (m *Log) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *Log) GetData() []*common.KeyStringValuePair {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*SegmentObject)(nil), "SegmentObject")
	proto.RegisterType((*SegmentReference)(nil), "SegmentReference")
	proto.RegisterType((*SpanObjectV2)(nil), "SpanObjectV2")
	proto.RegisterType((*Log)(nil), "Log")
}

func init() { proto.RegisterFile("language-agent-v2/trace.proto", fileDescriptor_8124ab206744863a) }

var fileDescriptor_8124ab206744863a = []byte{
	// 779 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x95, 0xdf, 0x8f, 0xdb, 0x44,
	0x10, 0xc7, 0xf1, 0x9d, 0xef, 0x87, 0xe7, 0x2e, 0x69, 0xba, 0x45, 0xc5, 0x3d, 0x81, 0x14, 0x05,
	0x0a, 0x51, 0xc5, 0x6d, 0x84, 0x2b, 0xa1, 0xbe, 0xf0, 0xd0, 0xa2, 0x0a, 0x45, 0x84, 0x12, 0x6d,
	0xae, 0x45, 0xe2, 0x6d, 0xcf, 0x9e, 0x73, 0x4d, 0xe2, 0x5d, 0xb3, 0xde, 0x5c, 0x65, 0x5e, 0x79,
	0xe7, 0x85, 0xff, 0x82, 0xff, 0x88, 0xff, 0xa6, 0xda, 0xb1, 0xf3, 0xc3, 0xc9, 0xdd, 0x53, 0x76,
	0x3e, 0xdf, 0xc9, 0x7a, 0x67, 0xbe, 0x9e, 0x35, 0x7c, 0xb1, 0x90, 0x2a, 0x5d, 0xca, 0x14, 0x2f,
	0x65, 0x8a, 0xca, 0x5e, 0xde, 0x46, 0x23, 0x6b, 0x64, 0x8c, 0xbc, 0x30, 0xda, 0xea, 0x8b, 0x47,
	0xb1, 0xce, 0x73, 0xad, 0x46, 0xf5, 0x4f, 0x03, 0x9f, 0x34, 0x90, 0x12, 0x2f, 0xb7, 0xa5, 0xc1,
	0xff, 0x1e, 0x74, 0x66, 0x98, 0xe6, 0xa8, 0xec, 0xaf, 0xd7, 0x7f, 0x60, 0x6c, 0xd9, 0x77, 0xd0,
	0xa5, 0xbc, 0x86, 0x8e, 0x93, 0xd0, 0xeb, 0x7b, 0xc3, 0xb3, 0x28, 0xe0, 0x6f, 0x55, 0xf6, 0xe7,
	0x12, 0xc7, 0x89, 0xd8, 0x49, 0x60, 0x5f, 0xc2, 0x51, 0x59, 0x48, 0x55, 0x86, 0x07, 0xfd, 0xc3,
	0xe1, 0x59, 0xd4, 0xe1, 0xb3, 0x42, 0xaa, 0x7a, 0xbb, 0x77, 0x91, 0xa8, 0x35, 0xf6, 0x39, 0x04,
	0x25, 0x9a, 0xdb, 0x2c, 0xc6, 0x71, 0x12, 0x1e, 0xf6, 0xbd, 0xe1, 0x91, 0xd8, 0x00, 0xf6, 0x2d,
	0x3c, 0x5c, 0x05, 0xaa, 0xb4, 0x52, 0x51, 0x96, 0x4f, 0x59, 0xfb, 0x02, 0xfb, 0x0a, 0x3a, 0x59,
	0x39, 0xcb, 0xfe, 0xc2, 0x49, 0x96, 0x67, 0x16, 0x93, 0xf0, 0xa8, 0xef, 0x0d, 0x4f, 0x45, 0x1b,
	0x0e, 0xfe, 0xf6, 0xa1, 0xd7, 0x1c, 0x52, 0xe0, 0x0d, 0x1a, 0x54, 0x31, 0xb2, 0x01, 0x9c, 0x18,
	0xbc, 0xb9, 0xaa, 0x0a, 0xa4, 0xba, 0xba, 0xd1, 0x29, 0x17, 0x75, 0x2c, 0x56, 0x02, 0xfb, 0x01,
	0x3e, 0x2d, 0xa4, 0x41, 0x65, 0xaf, 0xda, 0x8d, 0x38, 0xd8, 0x6d, 0xc4, 0x9d, 0x69, 0x6c, 0x00,
	0xe7, 0x35, 0x77, 0x6d, 0x58, 0x17, 0xdb, 0x62, 0xec, 0x05, 0x7c, 0xd6, 0xc4, 0xf7, 0x54, 0x7d,
	0x9f, 0xcc, 0xbe, 0x86, 0xae, 0x42, 0xfb, 0x41, 0x9b, 0xf9, 0xcb, 0x24, 0x31, 0x58, 0x96, 0x54,
	0x7c, 0x20, 0x76, 0x28, 0x7b, 0x06, 0xbd, 0x36, 0x19, 0x27, 0xe1, 0x31, 0x6d, 0xbd, 0xc7, 0xd9,
	0xf7, 0xf0, 0x18, 0x95, 0x35, 0xd5, 0xfe, 0x61, 0x4e, 0xe8, 0x1f, 0xf7, 0xa8, 0xce, 0x07, 0x52,
	0x5e, 0xab, 0xa4, 0xd0, 0x99, 0xb2, 0xe1, 0x29, 0x1d, 0xa5, 0x0d, 0xd9, 0x10, 0x1e, 0xb4, 0xc0,
	0x38, 0x09, 0x03, 0xda, 0x76, 0x17, 0xbb, 0xda, 0xea, 0xb2, 0xd7, 0x1b, 0x42, 0x5d, 0x5b, 0x9b,
	0xba, 0xda, 0xda, 0x64, 0x9c, 0x84, 0x67, 0x75, 0x6d, 0xbb, 0x7c, 0xf0, 0xaf, 0x0f, 0xe7, 0xdb,
	0xef, 0x23, 0x7b, 0x0c, 0xc7, 0x65, 0x6d, 0x8c, 0x47, 0x7f, 0x69, 0xa2, 0x3d, 0xdb, 0x0e, 0xee,
	0xb0, 0xcd, 0xbd, 0xc4, 0x56, 0x1a, 0x7b, 0x95, 0xe5, 0x48, 0xbe, 0x1e, 0x8a, 0x0d, 0x60, 0x21,
	0x9c, 0xa0, 0x4a, 0x48, 0xf3, 0x49, 0x5b, 0x85, 0xec, 0x29, 0xf8, 0x06, 0x6f, 0x9c, 0x55, 0x6e,
	0x40, 0x1e, 0xf2, 0xdd, 0xd7, 0x52, 0x90, 0xec, 0x3a, 0xa5, 0x0b, 0x34, 0xd2, 0x66, 0x5a, 0xbd,
	0x91, 0x39, 0xae, 0x2d, 0xdb, 0xc5, 0xae, 0xf3, 0x2d, 0x44, 0x46, 0x05, 0xa2, 0x0d, 0x5d, 0xa9,
	0x05, 0xa2, 0x19, 0x27, 0x64, 0xcc, 0x91, 0x68, 0x22, 0xc6, 0xc0, 0x77, 0x2b, 0xb2, 0x21, 0x10,
	0xb4, 0x66, 0x4f, 0xe1, 0xd4, 0x35, 0x82, 0x26, 0x03, 0x68, 0x32, 0x02, 0x9a, 0x63, 0x1a, 0x8d,
	0xb5, 0xc4, 0x86, 0x10, 0xb8, 0xf5, 0x44, 0x56, 0x68, 0xa8, 0xe7, 0xdd, 0x08, 0x28, 0x8f, 0x88,
	0xd8, 0x88, 0xac, 0x0f, 0x67, 0xb1, 0xce, 0x0b, 0xad, 0xea, 0xe1, 0x39, 0xa7, 0x13, 0x6c, 0x23,
	0xd7, 0xcd, 0x75, 0x18, 0x76, 0xe8, 0x2c, 0x1b, 0xe0, 0xba, 0x99, 0x95, 0xaf, 0x8d, 0xd1, 0x26,
	0xec, 0xd2, 0x78, 0xaf, 0x42, 0xf6, 0x0d, 0xf8, 0x56, 0xa6, 0x65, 0xf8, 0x80, 0xba, 0xf9, 0x88,
	0xff, 0x8c, 0xd5, 0xcc, 0x9a, 0x4c, 0xa5, 0xef, 0xe4, 0x62, 0x89, 0x53, 0x99, 0x19, 0x41, 0x09,
	0x2c, 0x04, 0x7f, 0xa1, 0xd3, 0x32, 0xec, 0x51, 0xa2, 0xcf, 0x27, 0x3a, 0x15, 0x44, 0x06, 0xaf,
	0xe0, 0x70, 0xa2, 0x53, 0xd7, 0x08, 0xeb, 0xec, 0xf2, 0xc8, 0x2e, 0x5a, 0xbb, 0xdd, 0x13, 0x69,
	0x65, 0x73, 0x99, 0xdd, 0xbd, 0xbb, 0x4b, 0x88, 0x7e, 0x82, 0x27, 0xdb, 0x93, 0x2f, 0xb0, 0xd0,
	0x66, 0x35, 0xb0, 0xec, 0x19, 0x9c, 0xc4, 0x7a, 0xb1, 0x70, 0x37, 0x6a, 0x8f, 0xbf, 0x2d, 0x4a,
	0x6b, 0x50, 0xe6, 0x4d, 0xe6, 0x45, 0xc0, 0x7f, 0xd4, 0x79, 0x2e, 0x55, 0x52, 0x0e, 0x3e, 0x19,
	0x7a, 0xaf, 0xfe, 0xf1, 0xe0, 0xb9, 0x36, 0x29, 0x97, 0x85, 0x8c, 0xdf, 0x23, 0x2f, 0xe7, 0xd5,
	0x07, 0xb9, 0x98, 0x67, 0xca, 0x91, 0x9c, 0x37, 0xd3, 0xca, 0x57, 0x17, 0x3f, 0xa7, 0x8b, 0x9f,
	0xdf, 0x46, 0x53, 0xef, 0xf7, 0x17, 0x69, 0x66, 0xdf, 0x2f, 0xaf, 0x79, 0xac, 0xf3, 0xd1, 0x6c,
	0x5e, 0xbd, 0x9c, 0xfe, 0x32, 0x4a, 0x75, 0x54, 0xce, 0xab, 0x91, 0xa1, 0xd3, 0xa0, 0x19, 0xa5,
	0xa6, 0x88, 0x47, 0x7b, 0x1f, 0x8d, 0xff, 0x0e, 0x2e, 0x66, 0xf3, 0xea, 0xb7, 0xe6, 0x31, 0x6f,
	0xea, 0x47, 0x4c, 0xdd, 0x27, 0x21, 0xd6, 0x8b, 0xeb, 0x63, 0xfa, 0x38, 0x3c, 0xff, 0x18, 0x00,
	0x00, 0xff, 0xff, 0x4a, 0x11, 0x04, 0x34, 0x6d, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// TraceSegmentReportServiceClient is the client API for TraceSegmentReportService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TraceSegmentReportServiceClient interface {
	Collect(ctx context.Context, opts ...grpc.CallOption) (TraceSegmentReportService_CollectClient, error)
}

type traceSegmentReportServiceClient struct {
	cc *grpc.ClientConn
}

func NewTraceSegmentReportServiceClient(cc *grpc.ClientConn) TraceSegmentReportServiceClient {
	return &traceSegmentReportServiceClient{cc}
}

func (c *traceSegmentReportServiceClient) Collect(ctx context.Context, opts ...grpc.CallOption) (TraceSegmentReportService_CollectClient, error) {
	stream, err := c.cc.NewStream(ctx, &_TraceSegmentReportService_serviceDesc.Streams[0], "/TraceSegmentReportService/collect", opts...)
	if err != nil {
		return nil, err
	}
	x := &traceSegmentReportServiceCollectClient{stream}
	return x, nil
}

type TraceSegmentReportService_CollectClient interface {
	Send(*common.UpstreamSegment) error
	CloseAndRecv() (*common.Commands, error)
	grpc.ClientStream
}

type traceSegmentReportServiceCollectClient struct {
	grpc.ClientStream
}

func (x *traceSegmentReportServiceCollectClient) Send(m *common.UpstreamSegment) error {
	return x.ClientStream.SendMsg(m)
}

func (x *traceSegmentReportServiceCollectClient) CloseAndRecv() (*common.Commands, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(common.Commands)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TraceSegmentReportServiceServer is the server API for TraceSegmentReportService service.
type TraceSegmentReportServiceServer interface {
	Collect(TraceSegmentReportService_CollectServer) error
}

func RegisterTraceSegmentReportServiceServer(s *grpc.Server, srv TraceSegmentReportServiceServer) {
	s.RegisterService(&_TraceSegmentReportService_serviceDesc, srv)
}

func _TraceSegmentReportService_Collect_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TraceSegmentReportServiceServer).Collect(&traceSegmentReportServiceCollectServer{stream})
}

type TraceSegmentReportService_CollectServer interface {
	SendAndClose(*common.Commands) error
	Recv() (*common.UpstreamSegment, error)
	grpc.ServerStream
}

type traceSegmentReportServiceCollectServer struct {
	grpc.ServerStream
}

func (x *traceSegmentReportServiceCollectServer) SendAndClose(m *common.Commands) error {
	return x.ServerStream.SendMsg(m)
}

func (x *traceSegmentReportServiceCollectServer) Recv() (*common.UpstreamSegment, error) {
	m := new(common.UpstreamSegment)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _TraceSegmentReportService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "TraceSegmentReportService",
	HandlerType: (*TraceSegmentReportServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "collect",
			Handler:       _TraceSegmentReportService_Collect_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "language-agent-v2/trace.proto",
}
