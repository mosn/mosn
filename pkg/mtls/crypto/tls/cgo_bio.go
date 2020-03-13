package tls

// #include "shim.h"
import "C"

import (
	"net"
	"reflect"
	"sync"
	"unsafe"
)

// SSLRecordSize is basic buf size of io-buffer
const (
	SSLRecordSize = 65536
)

func nonCopyGoBytes(ptr uintptr, length int) []byte {
	var slice []byte
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	header.Cap = length
	header.Len = length
	header.Data = ptr
	return slice
}

func nonCopyCString(data *C.char, size C.int) []byte {
	return nonCopyGoBytes(uintptr(unsafe.Pointer(data)), int(size))
}

// WriteBio is use to be the IO-writer of openssl-bio
type WriteBio struct {
	dataMtx sync.Mutex
	buf     []byte
	conn    net.Conn
	mtx     *sync.Mutex
	err     error
}

func loadWritePtr(b *C.BIO) *WriteBio {
	//t := token(C.X_BIO_get_data(b))
	return (*WriteBio)(C.X_BIO_get_data(b))
}

func bioClearRetryFlags(b *C.BIO) {
	C.X_BIO_clear_flags(b, C.BIO_FLAGS_RWS|C.BIO_FLAGS_SHOULD_RETRY)
}

func bioSetRetryRead(b *C.BIO) {
	C.X_BIO_set_flags(b, C.BIO_FLAGS_READ|C.BIO_FLAGS_SHOULD_RETRY)
}

//export go_write_bio_write
func go_write_bio_write(b *C.BIO, data *C.char, size C.int) (rc C.int) {
	ptr := loadWritePtr(b)
	if ptr == nil || data == nil || size < 0 {
		return -1
	}
	bioClearRetryFlags(b)
	ptr.err = nil
	sourceBuf := nonCopyCString(data, size)
	ptr.mtx.Unlock()
	n, err := ptr.conn.Write(sourceBuf)
	ptr.mtx.Lock()
	if err != nil {
		ptr.err = err
		return C.int(-1)
	}
	return C.int(n)
}

//export go_write_bio_ctrl
func go_write_bio_ctrl(b *C.BIO, cmd C.int, arg1 C.long, arg2 unsafe.Pointer) (
	rc C.long) {
	defer func() {
		if err := recover(); err != nil {
			if BabasslPrintTraceTag.IsOpen() {
				print("openssl: writeBioCtrl panic'd: %v", err)
			}
			rc = -1
		}
	}()
	switch cmd {
	case C.BIO_CTRL_WPENDING:
		return writeBioPending(b)
	case C.BIO_CTRL_DUP, C.BIO_CTRL_FLUSH:
		return 1
	default:
		return 0
	}
}

func writeBioPending(b *C.BIO) C.long {
	ptr := loadWritePtr(b)
	if ptr == nil {
		return 0
	}
	ptr.dataMtx.Lock()
	defer ptr.dataMtx.Unlock()
	return C.long(len(ptr.buf))
}

// MakeCBIO used to make and set real bio of openssl, then return
// go WriteBio for use
func (b *WriteBio) MakeCBIO() *C.BIO {
	rv := C.X_BIO_new_write_bio()
	C.BIO_set_data(rv, unsafe.Pointer(b))
	return rv
}

// ReadBio is use to be the IO-reader of openssl-bio
type ReadBio struct {
	dataMtx    sync.Mutex
	conn       net.Conn
	buf        []byte
	readIndex  int
	writeIndex int
	notClear   bool
	eof        bool
	mtx        *sync.Mutex
	err        error
}

func loadReadPtr(b *C.BIO) *ReadBio {
	return (*ReadBio)(C.X_BIO_get_data(b))
}

//export go_read_bio_read
func go_read_bio_read(b *C.BIO, data *C.char, size C.int) (rc C.int) {

	ptr := loadReadPtr(b)
	if ptr == nil || size < 0 {
		return C.int(-1)
	}

	bioClearRetryFlags(b)
	ptr.err = nil
	conn := ptr.conn
	targetBuf := nonCopyCString(data, size)
	ptr.mtx.Unlock()
	n, err := conn.Read(targetBuf)
	ptr.mtx.Lock()
	//for raw bytes
	if cap(ptr.buf) < n {
		ptr.buf = make([]byte, n+SSLRecordSize)
	}
	copy(ptr.buf, targetBuf)
	if err != nil {
		ptr.err = err
		return C.int(-1)
	}

	return C.int(n)
}

//export go_read_bio_ctrl
func go_read_bio_ctrl(b *C.BIO, cmd C.int, arg1 C.long, arg2 unsafe.Pointer) (
	rc C.long) {

	defer func() {
		if err := recover(); err != nil {
			if BabasslPrintTraceTag.IsOpen() {
				print("openssl: readBioCtrl panic'd: %v", err)
			}
			rc = -1
		}
	}()
	switch cmd {
	case C.BIO_CTRL_PENDING:
		return readBioPending(b)
	case C.BIO_CTRL_DUP, C.BIO_CTRL_FLUSH:
		return 1
	default:
		return 0
	}
}

func readBioPending(b *C.BIO) C.long {
	ptr := loadReadPtr(b)
	if ptr == nil {
		return 0
	}
	ptr.dataMtx.Lock()
	defer ptr.dataMtx.Unlock()
	return C.long(len(ptr.buf))
}

func (b *ReadBio) getRawInput() []byte {
	return b.buf
}

// MakeCBIO used to make and set real bio of openssl, then return
// go ReadBio for use
func (b *ReadBio) MakeCBIO() *C.BIO {
	rv := C.X_BIO_new_read_bio()
	C.X_BIO_set_data(rv, unsafe.Pointer(b))
	return rv
}
