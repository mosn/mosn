package wasmer

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	wasmerGo "github.com/wasmerio/wasmer-go/wasmer"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/types"
)

func TestRegisterFunc(t *testing.T) {
	vm := NewWasmerVM()
	assert.Equal(t, vm.Name(), "wasmer")

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

func TestInstanceMalloc(t *testing.T) {
	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`
			(module
				(func (export "_start"))
				(func (export "malloc") (param i32) (result i32) i32.const 10))
	`))
	ins := module.NewInstance()

	assert.Nil(t, ins.RegisterFunc("TestRegisterFuncRecover", "somePanic", func(instance types.WasmInstance) int32 {
		panic("some panic")
	}))

	assert.Nil(t, ins.Start())

	addr, err := ins.Malloc(100)
	assert.Nil(t, err)
	assert.Equal(t, addr, uint64(10))
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

	assert.Nil(t, ins.PutMemory(uint64(300), 10, []byte("1111111111")))
	bs, err := ins.GetMemory(uint64(300), 10)
	assert.Nil(t, err)
	assert.Equal(t, string(bs), "1111111111")
}

func TestInstanceData(t *testing.T) {
	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`
			(module
				(func (export "_start")))
	`))
	ins := module.NewInstance()
	assert.Nil(t, ins.Start())

	var data int = 1
	ins.SetData(data)
	assert.Equal(t, ins.GetData().(int), 1)

	for i := 0; i < 10; i++ {
		ins.Lock(i)
		assert.Equal(t, ins.GetData().(int), i)
		ins.Unlock()
	}
}

func TestWasmerTypes(t *testing.T) {
	testDatas := []struct {
		refType     reflect.Type
		refValue    reflect.Value
		refValKind  reflect.Kind
		wasmValKind wasmerGo.ValueKind
	}{
		{reflect.TypeOf(int32(0)), reflect.ValueOf(int32(0)), reflect.Int32, wasmerGo.I32},
		{reflect.TypeOf(int64(0)), reflect.ValueOf(int64(0)), reflect.Int64, wasmerGo.I64},
		{reflect.TypeOf(float32(0)), reflect.ValueOf(float32(0)), reflect.Float32, wasmerGo.F32},
		{reflect.TypeOf(float64(0)), reflect.ValueOf(float64(0)), reflect.Float64, wasmerGo.F64},
	}

	for _, tc := range testDatas {
		assert.Equal(t, convertFromGoType(tc.refType).Kind(), tc.wasmValKind)
		assert.Equal(t, convertToGoTypes(convertFromGoValue(tc.refValue)).Kind(), tc.refValKind)
	}
}

func TestRefCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	destroyCount := 0
	abi := mock.NewMockABI(ctrl)
	abi.EXPECT().OnInstanceDestroy(gomock.Any()).DoAndReturn(func(types.WasmInstance) {
		destroyCount++
	})

	vm := NewWasmerVM()
	module := vm.NewModule([]byte(`(module (func (export "_start")))`))
	ins := NewWasmerInstance(vm.(*VM), module.(*Module))

	ins.abiList = []types.ABI{abi}

	assert.False(t, ins.Acquire())

	ins.started = 1
	for i := 0; i < 100; i++ {
		assert.True(t, ins.Acquire())
	}
	assert.Equal(t, ins.refCount, 100)

	ins.Stop()
	ins.Stop() // double stop
	time.Sleep(time.Second)
	assert.Equal(t, ins.started, uint32(1))

	for i := 0; i < 100; i++ {
		ins.Release()
	}

	time.Sleep(time.Second)
	assert.False(t, ins.Acquire())
	assert.Equal(t, ins.started, uint32(0))
	assert.Equal(t, ins.refCount, 0)
	assert.Equal(t, destroyCount, 1)
}
