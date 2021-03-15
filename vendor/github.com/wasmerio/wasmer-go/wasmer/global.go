package wasmer

// #include <wasmer_wasm.h>
import "C"
import (
	"runtime"
)

// Global stores a single value of the given GlobalType.
//
// See also
//
// https://webassembly.github.io/spec/core/syntax/modules.html#globals
//
type Global struct {
	_inner   *C.wasm_global_t
	_ownedBy interface{}
}

func newGlobal(pointer *C.wasm_global_t, ownedBy interface{}) *Global {
	global := &Global{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(global, func(global *Global) {
			C.wasm_global_delete(global.inner())
		})
	}

	return global
}

// NewGlobal instantiates a new Global in the given Store.
//
// It takes three arguments, the Store, the GlobalType and the Value for the Global.
//
//   valueType := NewValueType(I32)
//   globalType := NewGlobalType(valueType, CONST)
//   global := NewGlobal(store, globalType, NewValue(42, I32))
//
func NewGlobal(store *Store, ty *GlobalType, value Value) *Global {
	pointer := C.wasm_global_new(
		store.inner(),
		ty.inner(),
		value.inner(),
	)

	return newGlobal(pointer, nil)
}

func (self *Global) inner() *C.wasm_global_t {
	return self._inner
}

func (self *Global) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// IntoExtern converts the Global into an Extern.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   extern := global.IntoExtern()
//
func (self *Global) IntoExtern() *Extern {
	pointer := C.wasm_global_as_extern(self.inner())

	return newExtern(pointer, self.ownedBy())
}

// Type returns the Global's GlobalType.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   ty := global.Type()
//
func (self *Global) Type() *GlobalType {
	ty := C.wasm_global_type(self.inner())

	runtime.KeepAlive(self)

	return newGlobalType(ty, self.ownedBy())
}

// Set sets the Global's value.
//
// It takes two arguments, the Global's value as a native Go value and the value's ValueKind.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   _ = global.Set(1, I32)
//
func (self *Global) Set(value interface{}, kind ValueKind) error {
	if self.Type().Mutability() == IMMUTABLE {
		return newErrorWith("The global variable is not mutable, cannot set a new value")
	}

	result, err := fromGoValue(value, kind)

	if err != nil {
		//TODO: Make this error explicit
		panic(err.Error())
	}

	C.wasm_global_set(self.inner(), &result)

	return nil
}

// Get returns the Global's value as a native Go value.
//
//   global, _ := instance.Exports.GetGlobal("exported_global")
//   value, _ := global.Get()
//
func (self *Global) Get() (interface{}, error) {
	var value C.wasm_val_t

	C.wasm_global_get(self.inner(), &value)

	return toGoValue(&value), nil
}
