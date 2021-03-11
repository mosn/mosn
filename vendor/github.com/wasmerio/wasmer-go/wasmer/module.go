package wasmer

// #include <wasmer_wasm.h>
//
// #define own
//
// // We can't create a `wasm_byte_vec_t` directly in Go otherwise cgo
// // complains with “Go pointer to Go pointer”. The hack consists at
// // creating the `wasm_byte_vec_t` directly in C.
//
// own wasm_module_t* to_wasm_module_new(wasm_store_t *store, uint8_t *bytes, size_t bytes_length) {
//     wasm_byte_vec_t wasm_bytes;
//     wasm_bytes.size = bytes_length;
//     wasm_bytes.data = (wasm_byte_t*) bytes;
//
//     return wasm_module_new(store, &wasm_bytes);
// }
//
// bool to_wasm_module_validate(wasm_store_t *store, uint8_t *bytes, size_t bytes_length) {
//     wasm_byte_vec_t wasm_bytes;
//     wasm_bytes.size = bytes_length;
//     wasm_bytes.data = (wasm_byte_t*) bytes;
//
//     return wasm_module_validate(store, &wasm_bytes);
// }
//
// wasm_module_t* to_wasm_module_deserialize(wasm_store_t *store, uint8_t *bytes, size_t bytes_length) {
//     wasm_byte_vec_t serialized_bytes;
//     serialized_bytes.size = bytes_length;
//     serialized_bytes.data = (wasm_byte_t*) bytes;
//
//     return wasm_module_deserialize(store, &serialized_bytes);
// }
import "C"
import (
	"runtime"
	"unsafe"
)

// Module contains stateless WebAssembly code that has already been
// compiled and can be instantiated multiple times.
//
// WebAssembly programs are organized into modules, which are the unit
// of deployment, loading, and compilation. A module collects
// definitions for types, functions, tables, memories, and globals. In
// addition, it can declare imports and exports and provide
// initialization logic in the form of data and element segments or a
// start function.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/modules.html#modules
type Module struct {
	_inner *C.wasm_module_t
	store  *Store
}

// NewModule instantiates a new Module with the given Store.
//
// It takes two arguments, the Store and the Wasm module as a byte
// array of WAT code.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, err := wasmer.NewModule(store, wasmBytes)
func NewModule(store *Store, bytes []byte) (*Module, error) {
	wasmBytes, err := Wat2Wasm(string(bytes))

	if err != nil {
		return nil, err
	}

	var wasmBytesPtr *C.uint8_t
	wasmBytesLength := len(wasmBytes)

	if wasmBytesLength > 0 {
		wasmBytesPtr = (*C.uint8_t)(unsafe.Pointer(&wasmBytes[0]))
	}

	var self *Module

	err2 := maybeNewErrorFromWasmer(func() bool {
		self = &Module{
			_inner: C.to_wasm_module_new(store.inner(), wasmBytesPtr, C.size_t(wasmBytesLength)),
			store:  store,
		}

		return self._inner == nil
	})

	if err2 != nil {
		return nil, err2
	}

	runtime.KeepAlive(bytes)
	runtime.KeepAlive(wasmBytes)
	runtime.SetFinalizer(self, func(self *Module) {
		C.wasm_module_delete(self.inner())
	})

	return self, nil
}

// ValidateModule validates a new Module against the given Store.
//
// It takes two arguments, the Store and the WebAssembly module as a
// byte array. The function returns an error describing why the bytes
// are invalid, otherwise it returns nil.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   err := wasmer.ValidateModule(store, wasmBytes)
//
//   isValid := err != nil
func ValidateModule(store *Store, bytes []byte) error {
	wasmBytes, err := Wat2Wasm(string(bytes))

	if err != nil {
		return err
	}

	var wasmBytesPtr *C.uint8_t
	wasmBytesLength := len(wasmBytes)

	if wasmBytesLength > 0 {
		wasmBytesPtr = (*C.uint8_t)(unsafe.Pointer(&wasmBytes[0]))
	}

	err2 := maybeNewErrorFromWasmer(func() bool {
		return false == C.to_wasm_module_validate(store.inner(), wasmBytesPtr, C.size_t(wasmBytesLength))
	})

	if err2 != nil {
		return err2
	}

	runtime.KeepAlive(bytes)
	runtime.KeepAlive(wasmBytes)

	return nil
}

func (self *Module) inner() *C.wasm_module_t {
	return self._inner
}

// Name returns the Module's name.
//
// Note:️ This is not part of the standard Wasm C API. It is Wasmer specific.
//
//   wasmBytes := []byte(`(module $moduleName)`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, _ := wasmer.NewModule(store, wasmBytes)
//   name := module.Name()
func (self *Module) Name() string {
	var name C.wasm_name_t

	C.wasmer_module_name(self.inner(), &name)

	goName := nameToString(&name)

	C.wasm_name_delete(&name)

	return goName
}

// Imports returns the Module's imports as an ImportType array.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, _ := wasmer.NewModule(store, wasmBytes)
//   imports := module.Imports()
func (self *Module) Imports() []*ImportType {
	return newImportTypes(self).importTypes
}

// Exports returns the Module's exports as an ExportType array.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, _ := wasmer.NewModule(store, wasmBytes)
//   exports := module.Exports()
func (self *Module) Exports() []*ExportType {
	return newExportTypes(self).exportTypes
}

// Serialize serializes the module and returns the Wasm code as an byte array.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, _ := wasmer.NewModule(store, wasmBytes)
//   bytes := module.Serialize()
func (self *Module) Serialize() ([]byte, error) {
	var bytes C.wasm_byte_vec_t

	err := maybeNewErrorFromWasmer(func() bool {
		C.wasm_module_serialize(self.inner(), &bytes)

		return bytes.data == nil
	})

	if err != nil {
		return nil, err
	}

	goBytes := C.GoBytes(unsafe.Pointer(bytes.data), C.int(bytes.size))
	C.wasm_byte_vec_delete(&bytes)

	return goBytes, nil
}

// DeserializeModule deserializes an byte array to a Module.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   module, _ := wasmer.NewModule(store, wasmBytes)
//   bytes := module.Serialize()
//   //...
//   deserializedModule := wasmer.DeserializeModule(store, bytes)
func DeserializeModule(store *Store, bytes []byte) (*Module, error) {
	var bytesPtr *C.uint8_t
	bytesLength := len(bytes)

	if bytesLength > 0 {
		bytesPtr = (*C.uint8_t)(unsafe.Pointer(&bytes[0]))
	}

	var self *Module

	err := maybeNewErrorFromWasmer(func() bool {
		self = &Module{
			_inner: C.to_wasm_module_deserialize(store.inner(), bytesPtr, C.size_t(bytesLength)),
		}

		return self._inner == nil
	})

	if err != nil {
		return nil, err
	}

	return self, nil
}
