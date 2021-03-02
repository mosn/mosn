// +build !custom_wasmer_runtime

package wasmer

// #cgo CFLAGS: -I${SRCDIR}/packaged/include
// #cgo LDFLAGS: -lwasmer
//
// #cgo linux,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-amd64 -L${SRCDIR}/packaged/lib/linux-amd64
// #cgo linux,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-aarch64 -L${SRCDIR}/packaged/lib/linux-aarch64
// #cgo darwin,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/darwin-amd64 -L${SRCDIR}/packaged/lib/darwin-amd64
//
// #include <wasmer_wasm.h>
import "C"

// See https://github.com/golang/go/issues/26366.
import (
	_ "github.com/wasmerio/wasmer-go/wasmer/packaged/include"
	_ "github.com/wasmerio/wasmer-go/wasmer/packaged/lib"
)
