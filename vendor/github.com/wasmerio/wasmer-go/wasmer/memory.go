package wasmer

// #include <wasmer_wasm.h>
import "C"
import (
	"reflect"
	"runtime"
	"unsafe"
)

// Memory is a vector of raw uninterpreted bytes.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/modules.html#memories
//
type Memory struct {
	_inner   *C.wasm_memory_t
	_ownedBy interface{}
}

func newMemory(pointer *C.wasm_memory_t, ownedBy interface{}) *Memory {
	memory := &Memory{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(memory, func(memory *Memory) {
			C.wasm_memory_delete(memory.inner())
		})
	}

	return memory
}

// NewMemory instantiates a new Memory in the given Store.
//
// It takes two arguments, the Store and the MemoryType for the Memory.
//
//   memory := wasmer.NewFunction(
//		store,
//		wasmer.NewMemoryType(wasmer.NewLimits(1, 4)),
//	 )
//
func NewMemory(store *Store, ty *MemoryType) *Memory {
	pointer := C.wasm_memory_new(store.inner(), ty.inner())

	runtime.KeepAlive(store)
	runtime.KeepAlive(ty)

	return newMemory(pointer, nil)
}

func (self *Memory) inner() *C.wasm_memory_t {
	return self._inner
}

func (self *Memory) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Type returns the Memory's MemoryType.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   ty := memory.Type()
//
func (self *Memory) Type() *MemoryType {
	ty := C.wasm_memory_type(self.inner())

	runtime.KeepAlive(self)

	return newMemoryType(ty, self.ownedBy())
}

// Size returns the Memory's size as Pages.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   size := memory.Size()
//
func (self *Memory) Size() Pages {
	return Pages(C.wasm_memory_size(self.inner()))
}

// Size returns the Memory's size as a number of bytes.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   size := memory.DataSize()
//
func (self *Memory) DataSize() uint {
	return uint(C.wasm_memory_data_size(self.inner()))
}

// Data returns the Memory's contents as an byte array.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   data := memory.Data()
//
func (self *Memory) Data() []byte {
	length := int(self.DataSize())
	data := (*C.byte_t)(C.wasm_memory_data(self.inner()))

	runtime.KeepAlive(self)

	var header reflect.SliceHeader
	header = *(*reflect.SliceHeader)(unsafe.Pointer(&header))

	header.Data = uintptr(unsafe.Pointer(data))
	header.Len = length
	header.Cap = length

	return *(*[]byte)(unsafe.Pointer(&header))
}

// Grow grows the Memory's size by a given number of Pages (the delta).
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   grown := memory.Grow(2)
//
func (self *Memory) Grow(delta Pages) bool {
	return bool(C.wasm_memory_grow(self.inner(), C.wasm_memory_pages_t(delta)))
}

// IntoExtern converts the Memory into an Extern.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   extern := memory.IntoExtern()
//
func (self *Memory) IntoExtern() *Extern {
	pointer := C.wasm_memory_as_extern(self.inner())

	return newExtern(pointer, self.ownedBy())
}
