package wasmer

// #include <wasmer_wasm.h>
import "C"
import (
	"unsafe"
)

// Wat2Wasm parsers a string as either WAT code or a binary Wasm module.
//
// ⚠️ This is not part of the standard Wasm C API. It is Wasmer specific.
//
//   wat := "(module)"
//   wasm, _ := Wat2Wasm(wat)
//   engine := wasmer.NewEngine()
//	 store := wasmer.NewStore(engine)
//	 module, _ := wasmer.NewModule(store, wasmBytes)
//
func Wat2Wasm(wat string) ([]byte, error) {
	var watBytes C.wasm_byte_vec_t
	var watLength = len(wat)

	C.wasm_byte_vec_new(&watBytes, C.size_t(watLength), C.CString(wat))
	defer C.wasm_byte_vec_delete(&watBytes)

	var wasm C.wasm_byte_vec_t

	err := maybeNewErrorFromWasmer(func() bool {
		C.wat2wasm(&watBytes, &wasm)

		return wasm.data == nil
	})

	if err != nil {
		return nil, err
	}

	defer C.wasm_byte_vec_delete(&wasm)

	wasmBytes := C.GoBytes(unsafe.Pointer(wasm.data), C.int(wasm.size))

	return wasmBytes, nil
}
