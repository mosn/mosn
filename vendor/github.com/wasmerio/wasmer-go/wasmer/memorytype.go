package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

// MemoryType classifies linear memories and their size range.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/types.html#memory-types
//
type MemoryType struct {
	_inner   *C.wasm_memorytype_t
	_ownedBy interface{}
}

func newMemoryType(pointer *C.wasm_memorytype_t, ownedBy interface{}) *MemoryType {
	memoryType := &MemoryType{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(memoryType, func(memoryType *MemoryType) {
			C.wasm_memorytype_delete(memoryType.inner())
		})
	}

	return memoryType
}

// NewMemoryType instantiates a new MemoryType given some Limits.
//
//   limits := NewLimits(1, 4)
//   memoryType := NewMemoryType(limits)
//
func NewMemoryType(limits *Limits) *MemoryType {
	pointer := C.wasm_memorytype_new(limits.inner())

	return newMemoryType(pointer, nil)
}

func (self *MemoryType) inner() *C.wasm_memorytype_t {
	return self._inner
}

func (self *MemoryType) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Limits returns the MemoryType's Limits.
//
//   limits := NewLimits(1, 4)
//   memoryType := NewMemoryType(limits)
//   _ = memoryType.Limits()
//
func (self *MemoryType) Limits() *Limits {
	limits := newLimits(C.wasm_memorytype_limits(self.inner()), self.ownedBy())

	runtime.KeepAlive(self)

	return limits
}

// IntoExternType converts the MemoryType into an ExternType.
//
//   limits := NewLimits(1, 4)
//   memoryType := NewMemoryType(limits)
//   externType = memoryType.IntoExternType()
//
func (self *MemoryType) IntoExternType() *ExternType {
	pointer := C.wasm_memorytype_as_externtype_const(self.inner())

	return newExternType(pointer, self.ownedBy())
}
