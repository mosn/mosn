package tools

import "unsafe"

//ByteToInt8 convert []byte to []int8
func ByteToInt8(s []byte) []int8 {
	d := *(*[]int8)(unsafe.Pointer(&s))
	return d
}

//Int8ToByte convert []int8 to []byte
func Int8ToByte(s []int8) []byte {
	d := *(*[]byte)(unsafe.Pointer(&s))
	return d
}
