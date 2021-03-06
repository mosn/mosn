// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/type/matcher/v3/string.proto

package envoy_type_matcher_v3

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/annotations"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
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

type StringMatcher struct {
	// Types that are valid to be assigned to MatchPattern:
	//	*StringMatcher_Exact
	//	*StringMatcher_Prefix
	//	*StringMatcher_Suffix
	//	*StringMatcher_HiddenEnvoyDeprecatedRegex
	//	*StringMatcher_SafeRegex
	MatchPattern         isStringMatcher_MatchPattern `protobuf_oneof:"match_pattern"`
	IgnoreCase           bool                         `protobuf:"varint,6,opt,name=ignore_case,json=ignoreCase,proto3" json:"ignore_case,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *StringMatcher) Reset()         { *m = StringMatcher{} }
func (m *StringMatcher) String() string { return proto.CompactTextString(m) }
func (*StringMatcher) ProtoMessage()    {}
func (*StringMatcher) Descriptor() ([]byte, []int) {
	return fileDescriptor_e33cffa01bf36e0e, []int{0}
}

func (m *StringMatcher) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StringMatcher.Unmarshal(m, b)
}
func (m *StringMatcher) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StringMatcher.Marshal(b, m, deterministic)
}
func (m *StringMatcher) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StringMatcher.Merge(m, src)
}
func (m *StringMatcher) XXX_Size() int {
	return xxx_messageInfo_StringMatcher.Size(m)
}
func (m *StringMatcher) XXX_DiscardUnknown() {
	xxx_messageInfo_StringMatcher.DiscardUnknown(m)
}

var xxx_messageInfo_StringMatcher proto.InternalMessageInfo

type isStringMatcher_MatchPattern interface {
	isStringMatcher_MatchPattern()
}

type StringMatcher_Exact struct {
	Exact string `protobuf:"bytes,1,opt,name=exact,proto3,oneof"`
}

type StringMatcher_Prefix struct {
	Prefix string `protobuf:"bytes,2,opt,name=prefix,proto3,oneof"`
}

type StringMatcher_Suffix struct {
	Suffix string `protobuf:"bytes,3,opt,name=suffix,proto3,oneof"`
}

type StringMatcher_HiddenEnvoyDeprecatedRegex struct {
	HiddenEnvoyDeprecatedRegex string `protobuf:"bytes,4,opt,name=hidden_envoy_deprecated_regex,json=hiddenEnvoyDeprecatedRegex,proto3,oneof"`
}

type StringMatcher_SafeRegex struct {
	SafeRegex *RegexMatcher `protobuf:"bytes,5,opt,name=safe_regex,json=safeRegex,proto3,oneof"`
}

func (*StringMatcher_Exact) isStringMatcher_MatchPattern() {}

func (*StringMatcher_Prefix) isStringMatcher_MatchPattern() {}

func (*StringMatcher_Suffix) isStringMatcher_MatchPattern() {}

func (*StringMatcher_HiddenEnvoyDeprecatedRegex) isStringMatcher_MatchPattern() {}

func (*StringMatcher_SafeRegex) isStringMatcher_MatchPattern() {}

func (m *StringMatcher) GetMatchPattern() isStringMatcher_MatchPattern {
	if m != nil {
		return m.MatchPattern
	}
	return nil
}

func (m *StringMatcher) GetExact() string {
	if x, ok := m.GetMatchPattern().(*StringMatcher_Exact); ok {
		return x.Exact
	}
	return ""
}

func (m *StringMatcher) GetPrefix() string {
	if x, ok := m.GetMatchPattern().(*StringMatcher_Prefix); ok {
		return x.Prefix
	}
	return ""
}

func (m *StringMatcher) GetSuffix() string {
	if x, ok := m.GetMatchPattern().(*StringMatcher_Suffix); ok {
		return x.Suffix
	}
	return ""
}

// Deprecated: Do not use.
func (m *StringMatcher) GetHiddenEnvoyDeprecatedRegex() string {
	if x, ok := m.GetMatchPattern().(*StringMatcher_HiddenEnvoyDeprecatedRegex); ok {
		return x.HiddenEnvoyDeprecatedRegex
	}
	return ""
}

func (m *StringMatcher) GetSafeRegex() *RegexMatcher {
	if x, ok := m.GetMatchPattern().(*StringMatcher_SafeRegex); ok {
		return x.SafeRegex
	}
	return nil
}

func (m *StringMatcher) GetIgnoreCase() bool {
	if m != nil {
		return m.IgnoreCase
	}
	return false
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*StringMatcher) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*StringMatcher_Exact)(nil),
		(*StringMatcher_Prefix)(nil),
		(*StringMatcher_Suffix)(nil),
		(*StringMatcher_HiddenEnvoyDeprecatedRegex)(nil),
		(*StringMatcher_SafeRegex)(nil),
	}
}

type ListStringMatcher struct {
	Patterns             []*StringMatcher `protobuf:"bytes,1,rep,name=patterns,proto3" json:"patterns,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *ListStringMatcher) Reset()         { *m = ListStringMatcher{} }
func (m *ListStringMatcher) String() string { return proto.CompactTextString(m) }
func (*ListStringMatcher) ProtoMessage()    {}
func (*ListStringMatcher) Descriptor() ([]byte, []int) {
	return fileDescriptor_e33cffa01bf36e0e, []int{1}
}

func (m *ListStringMatcher) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListStringMatcher.Unmarshal(m, b)
}
func (m *ListStringMatcher) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListStringMatcher.Marshal(b, m, deterministic)
}
func (m *ListStringMatcher) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListStringMatcher.Merge(m, src)
}
func (m *ListStringMatcher) XXX_Size() int {
	return xxx_messageInfo_ListStringMatcher.Size(m)
}
func (m *ListStringMatcher) XXX_DiscardUnknown() {
	xxx_messageInfo_ListStringMatcher.DiscardUnknown(m)
}

var xxx_messageInfo_ListStringMatcher proto.InternalMessageInfo

func (m *ListStringMatcher) GetPatterns() []*StringMatcher {
	if m != nil {
		return m.Patterns
	}
	return nil
}

func init() {
	proto.RegisterType((*StringMatcher)(nil), "envoy.type.matcher.v3.StringMatcher")
	proto.RegisterType((*ListStringMatcher)(nil), "envoy.type.matcher.v3.ListStringMatcher")
}

func init() { proto.RegisterFile("envoy/type/matcher/v3/string.proto", fileDescriptor_e33cffa01bf36e0e) }

var fileDescriptor_e33cffa01bf36e0e = []byte{
	// 440 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0x4f, 0x8b, 0xd4, 0x30,
	0x18, 0xc6, 0x37, 0x9d, 0xed, 0xd8, 0xcd, 0xb0, 0xb0, 0x16, 0xff, 0x94, 0x01, 0xb5, 0x33, 0xb3,
	0x68, 0x41, 0x68, 0x61, 0xc7, 0xd3, 0x1e, 0xa3, 0xc2, 0x22, 0x2a, 0x4b, 0xc5, 0x73, 0x89, 0xed,
	0x3b, 0xb3, 0x01, 0x4d, 0x4a, 0x92, 0x2d, 0x9d, 0x9b, 0x47, 0x11, 0x4f, 0x1e, 0xfd, 0x24, 0x7e,
	0x01, 0xbf, 0x80, 0x1f, 0xc4, 0x83, 0x78, 0x90, 0x39, 0x49, 0x92, 0xee, 0xe0, 0x30, 0xdd, 0x5b,
	0xd3, 0xf7, 0xf9, 0x3d, 0xcf, 0xf3, 0x86, 0xe0, 0x29, 0xf0, 0x46, 0xac, 0x32, 0xbd, 0xaa, 0x21,
	0xfb, 0x40, 0x75, 0x79, 0x01, 0x32, 0x6b, 0xe6, 0x99, 0xd2, 0x92, 0xf1, 0x65, 0x5a, 0x4b, 0xa1,
	0x45, 0x78, 0xdb, 0x6a, 0x52, 0xa3, 0x49, 0x3b, 0x4d, 0xda, 0xcc, 0xc7, 0x93, 0x7e, 0x54, 0xc2,
	0x12, 0x5a, 0x47, 0x8e, 0x27, 0x97, 0x55, 0x4d, 0x33, 0xca, 0xb9, 0xd0, 0x54, 0x33, 0xc1, 0x55,
	0xd6, 0x80, 0x54, 0x4c, 0xf0, 0x8d, 0xf9, 0x78, 0xe6, 0x5c, 0xfe, 0xd7, 0x54, 0x50, 0x4b, 0x28,
	0xed, 0xa1, 0x13, 0xdd, 0x6d, 0xe8, 0x7b, 0x56, 0x51, 0x0d, 0xd9, 0xd5, 0x87, 0x1b, 0x4c, 0xff,
	0x78, 0xf8, 0xf0, 0x8d, 0xed, 0xfa, 0xca, 0x35, 0x08, 0xef, 0x60, 0x1f, 0x5a, 0x5a, 0xea, 0x08,
	0xc5, 0x28, 0x39, 0x38, 0xdb, 0xcb, 0xdd, 0x31, 0x9c, 0xe0, 0x61, 0x2d, 0x61, 0xc1, 0xda, 0xc8,
	0x33, 0x03, 0x72, 0x63, 0x4d, 0xf6, 0xa5, 0x17, 0xa3, 0xb3, 0xbd, 0xbc, 0x1b, 0x18, 0x89, 0xba,
	0x5c, 0x18, 0xc9, 0x60, 0x47, 0xe2, 0x06, 0xe1, 0x5b, 0x7c, 0xef, 0x82, 0x55, 0x15, 0xf0, 0xc2,
	0xd6, 0x2e, 0xae, 0xaa, 0x42, 0x55, 0xd8, 0xbd, 0xa3, 0x7d, 0x4b, 0x1e, 0xad, 0x89, 0x2f, 0x07,
	0xc9, 0xc7, 0xe0, 0xfb, 0xaf, 0xdf, 0x3f, 0x7d, 0x14, 0x19, 0x8b, 0xb1, 0x03, 0x9f, 0x1b, 0xee,
	0xd9, 0x06, 0xcb, 0x0d, 0x15, 0xbe, 0xc6, 0x58, 0xd1, 0x05, 0x74, 0x1e, 0x7e, 0x8c, 0x92, 0xd1,
	0xc9, 0x2c, 0xed, 0xbd, 0xf6, 0xd4, 0x12, 0xdd, 0xb6, 0x24, 0x58, 0x13, 0xff, 0x33, 0xf2, 0x8e,
	0x4c, 0xc0, 0x81, 0xb1, 0x70, 0x7e, 0x0f, 0xf0, 0x88, 0x2d, 0xb9, 0x90, 0x50, 0x94, 0x54, 0x41,
	0x34, 0x8c, 0x51, 0x12, 0xe4, 0xd8, 0xfd, 0x7a, 0x4a, 0x15, 0x9c, 0x3e, 0xfa, 0xf6, 0xe3, 0xd3,
	0xfd, 0x29, 0x8e, 0x7b, 0x22, 0xb6, 0xae, 0x93, 0xdc, 0xc2, 0x87, 0x76, 0x50, 0xd4, 0x54, 0x6b,
	0x90, 0x3c, 0x1c, 0xfc, 0x25, 0x68, 0xfa, 0x05, 0xe1, 0x9b, 0x2f, 0x99, 0xd2, 0xdb, 0x57, 0xff,
	0x02, 0x07, 0x9d, 0x4a, 0x45, 0x28, 0x1e, 0x24, 0xa3, 0x93, 0xe3, 0x6b, 0x76, 0xd8, 0xce, 0x30,
	0x4b, 0x7c, 0x45, 0x5e, 0x80, 0xf2, 0x0d, 0x7f, 0xfa, 0xd8, 0x14, 0x7c, 0x88, 0x8f, 0x7b, 0xf8,
	0x9d, 0x60, 0xf2, 0x04, 0xcf, 0x98, 0x70, 0x51, 0xb5, 0x14, 0xed, 0xaa, 0x3f, 0x95, 0x8c, 0x1c,
	0x75, 0x6e, 0x5e, 0xce, 0x39, 0x7a, 0x37, 0xb4, 0x4f, 0x68, 0xfe, 0x2f, 0x00, 0x00, 0xff, 0xff,
	0x78, 0x7b, 0x7f, 0x81, 0x03, 0x03, 0x00, 0x00,
}
