package wasmer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
)

func TestRegisterFunc(t *testing.T) {
	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`(module (func (export "_start")))`))
	ins := module.NewInstance()

	// invalid namespace
	assert.Equal(t, ins.RegisterFunc("", "funcName", nil), ErrInvalidParam)

	// nil f
	assert.Equal(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", nil), ErrInvalidParam)

	var testStruct struct{}

	// f is not func
	assert.Equal(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", &testStruct), ErrRegisterNotFunc)

	// f is func with 0 args
	assert.Equal(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", func() {}), ErrRegisterArgNum)

	// f is func, but the first arg is not types.WasmInstance
	assert.Equal(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", func(first int32) {}), ErrRegisterArgType)

	assert.Nil(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", func(f types.WasmInstance) {}))

	assert.Nil(t, ins.Start())

	assert.Equal(t, ins.RegisterFunc("TestRegisterFuncNamespace", "funcName", func(f types.WasmInstance) {}), ErrInstanceAlreadyStart)
}

func TestRegisterFuncRecoverPanic(t *testing.T) {
	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`
			(module
				(import "TestRegisterFuncRecover" "somePanic" (func $somePanic (result i32)))
				(func (export "_start"))
				(func (export "panicTrigger") (param) (result i32)
					call $somePanic))
	`))
	ins := module.NewInstance()

	assert.Nil(t, ins.RegisterFunc("TestRegisterFuncRecover", "somePanic", func(instance types.WasmInstance) int32 {
		panic("some panic")
	}))

	assert.Nil(t, ins.Start())

	f, err := ins.GetExportsFunc("panicTrigger")
	assert.Nil(t, err)

	_, err = f.Call()
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "panic [some panic] when calling func [somePanic]")
}

func TestInstanceMem(t *testing.T) {
	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`(module (memory (export "memory") 1) (func (export "_start")))`))
	ins := module.NewInstance()
	assert.Nil(t, ins.Start())

	m, err := ins.GetExportsMem("memory")
	assert.Nil(t, err)
	// A WebAssembly page has a constant size of 65,536 bytes, i.e., 64KiB
	assert.Equal(t, len(m), 1<<16)

	assert.Nil(t, ins.PutByte(uint64(100), 'a'))
	b, err := ins.GetByte(uint64(100))
	assert.Nil(t, err)
	assert.Equal(t, b, byte('a'))

	assert.Nil(t, ins.PutUint32(uint64(200), 99))
	u, err := ins.GetUint32(uint64(200))
	assert.Nil(t, err)
	assert.Equal(t, u, uint32(99))
}
