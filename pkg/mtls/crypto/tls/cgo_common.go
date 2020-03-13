package tls

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"
)

type babasslTag struct {
	tag bool
	mtx sync.Mutex
}

func (tag *babasslTag) Open() {
	tag.mtx.Lock()
	defer tag.mtx.Unlock()
	tag.tag = true
}

func (tag *babasslTag) Close() {
	tag.mtx.Lock()
	defer tag.mtx.Unlock()
	tag.tag = false
}

func (tag *babasslTag) IsOpen() bool {
	return tag.tag
}

// UseBabasslTag is used to determine whether use babassl, default close
var UseBabasslTag = &babasslTag{
	tag: true,
}

// OpenBabasslTag is used to open UseBabasslTag, when this tag is open
// tls processing will work in openssl
func OpenBabasslTag() {
	UseBabasslTag.Open()
}

// CloseBabasslTag is used to close UseBabasslTag, when this tag is close
// tls processing will work in go tls
func CloseBabasslTag() {
	UseBabasslTag.Close()
}

//BabasslPrintTraceTag is use to determine whether open print trace, default close
var BabasslPrintTraceTag = &babasslTag{
	tag: false,
}

// OpenBabasslPrintTraceTag is used to open UseBabasslTag, when this tag is open
// some openssl debug infomation would print to stdout
func OpenBabasslPrintTraceTag() {
	BabasslPrintTraceTag.Open()
}

// CloseBabasslPrintTraceTag is used to close UseBabasslTag
func CloseBabasslPrintTraceTag() {
	BabasslPrintTraceTag.Close()
}

// BytesToString change []byte to string
func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}

// StringToBytes change string to []byte
func StringToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{sh.Data, sh.Len, 0}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

// PacketPeek2byteToLen would peek first 2 byte from packet, then set it to
// a uint16 variable
func PacketPeek2byteToLen(packet []byte) (uint16, error) {
	if len(packet) < 2 {
		return 0, errors.New("PacketPeek2byteToLen error, packet less than 2 byte")
	}

	res := uint16(packet[0]) << 8
	res |= uint16(packet[1])

	return res, nil
}

// PacketPeek1byteToLen would peek first 1 byte from packet, then set it to
// a uint8 variable
func PacketPeek1byteToLen(packet []byte) (uint8, error) {
	if len(packet) < 1 {
		return 0, errors.New("PacketPeek1byteToLen error, packet less than 1 byte")
	}

	res := uint8(packet[0])

	return res, nil
}
