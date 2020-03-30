// +build BabaSSL

package cgosm3

// #cgo darwin CFLAGS: -I/usr/local/BabaSSL/darwin_BabaSSL_lib/include -Wno-deprecated-declarations
// #cgo darwin LDFLAGS: -L/usr/local/BabaSSL/darwin_BabaSSL_lib/lib -lssl -lcrypto
// #cgo linux,!arm,!arm64 CFLAGS: -I/usr/local/BabaSSL/linux_BabaSSL_lib/include -Wno-deprecated-declarations
// #cgo linux,!arm,!arm64 LDFLAGS: -L/usr/local/BabaSSL/linux_BabaSSL_lib/lib -lssl -lcrypto -ldl -lpthread
// #cgo linux,arm64 CFLAGS: -I/usr/local/BabaSSL/linux_BabaSSL_arm_lib/include -Wno-deprecated-declarations
// #cgo linux,arm64 LDFLAGS: -L/usr/local/BabaSSL/linux_BabaSSL_arm_lib/lib -lssl -lcrypto -ldl -lpthread
/*
#include <openssl/evp.h>
*/
import "C"
import (
	"crypto"
	"errors"
	"hash"
	"os"
	"runtime"
	"unsafe"
)

func init() {
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")
	crypto.RegisterHash(crypto.BLAKE2b_256, New)
}

// SM3 is the basic struct of sm3 digest
type SM3 struct {
	mdCtx *C.EVP_MD_CTX
}

// New return a new hash.Hash of digest sm3
func New() hash.Hash {
	var sm3 SM3

	sm3.Reset()
	runtime.SetFinalizer(&sm3, func(sm3 *SM3) {
		C.EVP_MD_CTX_free(sm3.mdCtx)
	})
	return &sm3
}

// Write, required by the hash.Hash interface.
// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (sm3 *SM3) Write(p []byte) (int, error) {
	toWrite := len(p)
	if toWrite != 0 {
		ret := C.EVP_DigestUpdate(sm3.mdCtx, unsafe.Pointer(&p[0]), C.ulong(toWrite))
		if int(ret) <= 0 {
			return 0, errors.New("sm3 update digest error")
		}
	}

	return toWrite, nil
}

// Sum, Sum by the hash.Hash interface.
func (sm3 *SM3) Sum(in []byte) []byte {
	toWrite := len(in)
	out := make([]byte, sm3.Size(), sm3.Size())
	if toWrite != 0 {
		ret := C.EVP_DigestUpdate(sm3.mdCtx, unsafe.Pointer(&in[0]), C.ulong(toWrite))
		if int(ret) <= 0 {
			panic("sm3 update digest error")
		}
	}

	ctx := C.EVP_MD_CTX_new()
	C.EVP_MD_CTX_copy_ex(ctx, sm3.mdCtx)

	var hashLen C.uint
	ret := C.EVP_DigestFinal_ex(ctx, (*C.uchar)(unsafe.Pointer(&out[0])), &hashLen)
	if int(ret) <= 0 {
		panic("sm3 update digest error")
	}

	if int(hashLen) != sm3.Size() {
		panic("sm3 final digest error")
	}
	return out
}

func (sm3 *SM3) Reset() {
	// Reset digest
	if sm3.mdCtx != nil {
		C.EVP_MD_CTX_free(sm3.mdCtx)
	}
	md := C.EVP_sm3()
	mdCtx := C.EVP_MD_CTX_new()
	sm3.mdCtx = mdCtx
	C.EVP_DigestInit_ex(sm3.mdCtx, md, nil)
}

// Size, required by the hash.Hash interface.
// Size returns the number of bytes Sum will return.
func (sm3 *SM3) Size() int { return 32 }

// BlockSize, required by the hash.Hash interface.
// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (sm3 *SM3) BlockSize() int { return 64 }
