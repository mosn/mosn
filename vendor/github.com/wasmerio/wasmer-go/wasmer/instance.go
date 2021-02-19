package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

type Instance struct {
	_inner  *C.wasm_instance_t
	Exports *Exports
}

// NewInstance instantiates a new Instance.
//
// It takes two arguments, the Module and an ImportObject.
//
// ⚠️ Instantiating a module may return TrapError if the module's start function traps.
//
//   wasmBytes := []byte(`...`)
//   engine := wasmer.NewEngine()
//	 store := wasmer.NewStore(engine)
//	 module, err := wasmer.NewModule(store, wasmBytes)
//   importObject := wasmer.NewImportObject()
//   instance, err := wasmer.NewInstance(module, importObject)
//
func NewInstance(module *Module, imports *ImportObject) (*Instance, error) {
	var traps *C.wasm_trap_t
	externs, err := imports.intoInner(module)

	if err != nil {
		return nil, err
	}

	var instance *C.wasm_instance_t

	err2 := maybeNewErrorFromWasmer(func() bool {
		instance = C.wasm_instance_new(
			module.store.inner(),
			module.inner(),
			externs,
			&traps,
		)

		return traps == nil && instance == nil
	})

	if err2 != nil {
		return nil, err2
	}

	runtime.KeepAlive(module)
	runtime.KeepAlive(module.store)
	runtime.KeepAlive(imports)

	if traps != nil {
		return nil, newErrorFromTrap(traps)
	}

	self := &Instance{
		_inner:  instance,
		Exports: newExports(instance, module),
	}

	runtime.SetFinalizer(self, func(self *Instance) {
		C.wasm_instance_delete(self.inner())
	})

	return self, nil
}

func (self *Instance) inner() *C.wasm_instance_t {
	return self._inner
}
