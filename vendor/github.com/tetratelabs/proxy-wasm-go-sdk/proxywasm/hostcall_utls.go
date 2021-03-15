package proxywasm

import (
	"unsafe"
)

func stringBytePtr(msg string) *byte {
	if len(msg) == 0 {
		return nil
	}
	bt := *(*[]byte)(unsafe.Pointer(&msg))
	return &bt[0]
}
