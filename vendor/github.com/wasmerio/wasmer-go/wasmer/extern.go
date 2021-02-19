package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

type Extern struct {
	_inner   *C.wasm_extern_t
	_ownedBy interface{}
}

type IntoExtern interface {
	IntoExtern() *Extern
}

func newExtern(pointer *C.wasm_extern_t, ownedBy interface{}) *Extern {
	extern := &Extern{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(extern, func(extern *Extern) {
			C.wasm_extern_delete(extern.inner())
		})
	}

	return extern
}

func (self *Extern) inner() *C.wasm_extern_t {
	return self._inner
}

func (self *Extern) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

func (self *Extern) IntoExtern() *Extern {
	return self
}

// Kind returns the Extern's ExternKind.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   _ = global.IntoExtern().Kind()
//
func (self *Extern) Kind() ExternKind {
	kind := ExternKind(C.wasm_extern_kind(self.inner()))

	runtime.KeepAlive(self)

	return kind
}

// Type returns the Extern's ExternType.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   _ = global.IntoExtern().Type()
//
func (self *Extern) Type() *ExternType {
	ty := C.wasm_extern_type(self.inner())

	runtime.KeepAlive(self)

	return newExternType(ty, self.ownedBy())
}

// IntoFunction converts the Extern into a Function.
//
// ⚠️ If the Extern is not a Function, IntoFunction will return nil as its result.
//
//   function, _ := instance.Exports.GetFunction("exported_function")
//   extern = function.IntoExtern()
//   _ := extern.IntoFunction()
//
func (self *Extern) IntoFunction() *Function {
	pointer := C.wasm_extern_as_func(self.inner())

	if pointer == nil {
		return nil
	}

	return newFunction(pointer, nil, self.ownedBy())
}

// IntoGlobal converts the Extern into a Global.
//
// ⚠️ If the Extern is not a Global, IntoGlobal will return nil as its result.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   extern = global.IntoExtern()
//   _ := extern.IntoGlobal()
//
func (self *Extern) IntoGlobal() *Global {
	pointer := C.wasm_extern_as_global(self.inner())

	if pointer == nil {
		return nil
	}

	return newGlobal(pointer, self.ownedBy())
}

// IntoTable converts the Extern into a Table.
//
// ⚠️ If the Extern is not a Table, IntoTable will return nil as its result.
//
//   table, _ := instance.Exports.GetTable("exported_table")
//   extern = table.IntoExtern()
//   _ := extern.IntoTable()
//
func (self *Extern) IntoTable() *Table {
	pointer := C.wasm_extern_as_table(self.inner())

	if pointer == nil {
		return nil
	}

	return newTable(pointer, self.ownedBy())
}

// IntoMemory converts the Extern into a Memory.
//
// ⚠️ If the Extern is not a Memory, IntoMemory will return nil as its result.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   extern = memory.IntoExtern()
//   _ := extern.IntoMemory()
//
func (self *Extern) IntoMemory() *Memory {
	pointer := C.wasm_extern_as_memory(self.inner())

	if pointer == nil {
		return nil
	}

	return newMemory(pointer, self.ownedBy())
}
