package gxring

import "unsafe"

func GetLittleEndianUint32(b []byte, offset int) uint32 {
	return *(*uint32)(unsafe.Pointer(&b[offset]))
}

func PutLittleEndianUint32(b []byte, offset int, v uint32) {
	*(*uint32)(unsafe.Pointer(&b[offset])) = v
}

func GetLittleEndianUint64(b []byte, offset int) uint64 {
	return *(*uint64)(unsafe.Pointer(&b[offset]))
}

func PutLittleEndianUint64(b []byte, offset int, v uint64) {
	*(*uint64)(unsafe.Pointer(&b[offset])) = v
}
