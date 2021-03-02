package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

// Represents the kind of an Extern.
type ExternKind C.wasm_externkind_t

const (
	// Represents an extern of kind function.
	FUNCTION = ExternKind(C.WASM_EXTERN_FUNC)

	// Represents an extern of kind global.
	GLOBAL = ExternKind(C.WASM_EXTERN_GLOBAL)

	// Represents an extern of kind table.
	TABLE = ExternKind(C.WASM_EXTERN_TABLE)

	// Represents an extern of kind memory.
	MEMORY = ExternKind(C.WASM_EXTERN_MEMORY)
)

// String returns the ExternKind as a string.
//
//   FUNCTION.String() // "func"
//   GLOBAL.String()   // "global"
//   TABLE.String()    // "table"
//   MEMORY.String()   // "memory"
func (self ExternKind) String() string {
	switch self {
	case FUNCTION:
		return "func"
	case GLOBAL:
		return "global"
	case TABLE:
		return "table"
	case MEMORY:
		return "memory"
	}
	panic("Unknown extern kind") // unreachable
}

// ExternType classifies imports and external values with their respective types.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/types.html#external-types
type ExternType struct {
	_inner   *C.wasm_externtype_t
	_ownedBy interface{}
}

type IntoExternType interface {
	IntoExternType() *ExternType
}

func newExternType(pointer *C.wasm_externtype_t, ownedBy interface{}) *ExternType {
	externType := &ExternType{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(externType, func(externType *ExternType) {
			C.wasm_externtype_delete(externType.inner())
		})
	}

	return externType
}

func (self *ExternType) inner() *C.wasm_externtype_t {
	return self._inner
}

func (self *ExternType) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Kind returns the ExternType's ExternKind
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   extern = global.IntoExtern()
//   _ = extern.Kind()
func (self *ExternType) Kind() ExternKind {
	kind := ExternKind(C.wasm_externtype_kind(self.inner()))

	runtime.KeepAlive(self)

	return kind
}

// IntoFunctionType converts the ExternType into a FunctionType.
//
// Note:️ If the ExternType is not a FunctionType, IntoFunctionType
// will return nil as its result.
//
//   function, _ := instance.Exports.GetFunction("exported_function")
//   externType = function.IntoExtern().Type()
//   _ := externType.IntoFunctionType()
func (self *ExternType) IntoFunctionType() *FunctionType {
	pointer := C.wasm_externtype_as_functype_const(self.inner())

	if pointer == nil {
		return nil
	}

	return newFunctionType(pointer, self.ownedBy())
}

// IntoGlobalType converts the ExternType into a GlobalType.
//
// Note:️ If the ExternType is not a GlobalType, IntoGlobalType will
// return nil as its result.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   externType = global.IntoExtern().Type()
//   _ := externType.IntoGlobalType()
//
func (self *ExternType) IntoGlobalType() *GlobalType {
	pointer := C.wasm_externtype_as_globaltype_const(self.inner())

	if pointer == nil {
		return nil
	}

	return newGlobalType(pointer, self.ownedBy())
}

// IntoTableType converts the ExternType into a TableType.
//
// Note:️ If the ExternType is not a TableType, IntoTableType will
// return nil as its result.
//
//   table, _ := instance.Exports.GetTable("exported_table")
//   externType = table.IntoExtern().Type()
//   _ := externType.IntoTableType()
func (self *ExternType) IntoTableType() *TableType {
	pointer := C.wasm_externtype_as_tabletype_const(self.inner())

	if pointer == nil {
		return nil
	}

	return newTableType(pointer, self.ownedBy())
}

// IntoMemoryType converts the ExternType into a MemoryType.
//
// Note:️ If the ExternType is not a MemoryType, IntoMemoryType will
// return nil as its result.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   externType = memory.IntoExtern().Type()
//   _ := externType.IntoMemoryType()
//
func (self *ExternType) IntoMemoryType() *MemoryType {
	pointer := C.wasm_externtype_as_memorytype_const(self.inner())

	if pointer == nil {
		return nil
	}

	return newMemoryType(pointer, self.ownedBy())
}
