// +build wasmer

/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wasmer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"

	wasmerGo "github.com/wasmerio/wasmer-go/wasmer"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/pkg/utils"
)

var (
	ErrAddrOverflow         = errors.New("addr overflow")
	ErrInstanceNotStart     = errors.New("instance has not started")
	ErrInstanceAlreadyStart = errors.New("instance has already started")
	ErrInvalidParam         = errors.New("invalid param")
	ErrRegisterNotFunc      = errors.New("register a non-func object")
	ErrRegisterArgNum       = errors.New("register func with invalid arg num")
	ErrRegisterArgType      = errors.New("register func with invalid arg type")
)

type Instance struct {
	vm           *VM
	module       *Module
	importObject *wasmerGo.ImportObject
	instance     *wasmerGo.Instance
	debug        *dwarfInfo
	abiList      []types.ABI

	lock     sync.Mutex
	started  uint32
	refCount int
	stopCond *sync.Cond

	// for cache
	memory    *wasmerGo.Memory
	funcCache sync.Map // string -> *wasmerGo.Function

	// user-defined data
	data interface{}
}

type InstanceOptions func(instance *Instance)

func InstanceWithDebug(debug *dwarfInfo) InstanceOptions {
	return func(instance *Instance) {
		if debug != nil {
			instance.debug = debug
		}
	}
}

func NewWasmerInstance(vm *VM, module *Module, options ...InstanceOptions) *Instance {
	ins := &Instance{
		vm:     vm,
		module: module,
		lock:   sync.Mutex{},
	}
	ins.stopCond = sync.NewCond(&ins.lock)

	for _, option := range options {
		option(ins)
	}

	wasiEnv, err := wasmerGo.NewWasiStateBuilder("").Finalize()
	if err != nil || wasiEnv == nil {
		log.DefaultLogger.Warnf("[wasmer][instance] NewWasmerInstance fail to create wasi env, err: %v", err)

		ins.importObject = wasmerGo.NewImportObject()
		return ins
	}

	imo, err := wasiEnv.GenerateImportObject(ins.vm.store, ins.module.module)
	if err != nil {
		log.DefaultLogger.Warnf("[wasmer][instance] NewWasmerInstance fail to create import object, err: %v", err)

		ins.importObject = wasmerGo.NewImportObject()
	} else {
		ins.importObject = imo
	}

	return ins
}

func (w *Instance) GetData() interface{} {
	return w.data
}

func (w *Instance) SetData(data interface{}) {
	w.data = data
}

func (w *Instance) Acquire() bool {
	w.lock.Lock()
	defer w.lock.Unlock()

	if !w.checkStart() {
		return false
	}

	w.refCount++

	return true
}

func (w *Instance) Release() {
	w.lock.Lock()
	w.refCount--

	if w.refCount <= 0 {
		w.stopCond.Broadcast()
	}
	w.lock.Unlock()
}

func (w *Instance) Lock(data interface{}) {
	w.lock.Lock()
	w.data = data
}

func (w *Instance) Unlock() {
	w.data = nil
	w.lock.Unlock()
}

func (w *Instance) GetModule() types.WasmModule {
	return w.module
}

func (w *Instance) Start() error {
	w.abiList = abi.GetABIList(w)

	for _, abi := range w.abiList {
		abi.OnInstanceCreate(w)
	}

	ins, err := wasmerGo.NewInstance(w.module.module, w.importObject)
	if err != nil {
		log.DefaultLogger.Errorf("[wasmer][instance] Start fail to new wasmer-go instance, err: %v", err)
		return err
	}

	w.instance = ins

	f, err := w.instance.Exports.GetFunction("_start")
	if err != nil {
		log.DefaultLogger.Errorf("[wasmer][instance] Start fail to get export func: _start, err: %v", err)
		return err
	}

	_, err = f()
	if err != nil {
		log.DefaultLogger.Errorf("[wasmer][instance] Start fail to call _start func, err: %v", err)
		w.HandleError(err)
		return err
	}

	for _, abi := range w.abiList {
		abi.OnInstanceStart(w)
	}

	atomic.StoreUint32(&w.started, 1)

	return nil
}

func (w *Instance) Stop() {
	utils.GoWithRecover(func() {
		w.lock.Lock()
		for w.refCount > 0 {
			w.stopCond.Wait()
		}
		swapped := atomic.CompareAndSwapUint32(&w.started, 1, 0)
		w.lock.Unlock()

		if swapped {
			for _, abi := range w.abiList {
				abi.OnInstanceDestroy(w)
			}
		}
	}, nil)
}

// return true is Instance is started, false if not started.
func (w *Instance) checkStart() bool {
	return atomic.LoadUint32(&w.started) == 1
}

func (w *Instance) RegisterFunc(namespace string, funcName string, f interface{}) error {
	if w.checkStart() {
		log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc not allow to register func after instance started, namespace: %v, funcName: %v",
			namespace, funcName)
		return ErrInstanceAlreadyStart
	}

	if namespace == "" || funcName == "" {
		log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc invalid param, namespace: %v, funcName: %v", namespace, funcName)
		return ErrInvalidParam
	}

	if f == nil || reflect.ValueOf(f).IsNil() {
		log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc f is nil")
		return ErrInvalidParam
	}

	if reflect.TypeOf(f).Kind() != reflect.Func {
		log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc f is not func, actual type: %v", reflect.TypeOf(f))
		return ErrRegisterNotFunc
	}

	funcType := reflect.TypeOf(f)

	argsNum := funcType.NumIn()
	if argsNum < 1 {
		log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc invalid args num: %v, must >= 1", argsNum)
		return ErrRegisterArgNum
	}

	argsKind := make([]*wasmerGo.ValueType, argsNum-1)
	for i := 1; i < argsNum; i++ {
		argsKind[i-1] = convertFromGoType(funcType.In(i))
	}

	retsNum := funcType.NumOut()
	retsKind := make([]*wasmerGo.ValueType, retsNum)
	for i := 0; i < retsNum; i++ {
		retsKind[i] = convertFromGoType(funcType.Out(i))
	}

	fwasmer := wasmerGo.NewFunction(
		w.vm.store,
		wasmerGo.NewFunctionType(argsKind, retsKind),
		func(args []wasmerGo.Value) (callRes []wasmerGo.Value, err error) {
			defer func() {
				if r := recover(); r != nil {
					log.DefaultLogger.Errorf("[wasmer][instance] RegisterFunc recover func call: %v, r: %v, stack: %v",
						funcName, r, string(debug.Stack()))
					callRes = nil
					err = fmt.Errorf("panic [%v] when calling func [%v]", r, funcName)
				}
			}()

			callArgs := make([]reflect.Value, 1+len(args))
			callArgs[0] = reflect.ValueOf(w)

			for i, arg := range args {
				callArgs[i+1] = convertToGoTypes(arg)
			}

			callResult := reflect.ValueOf(f).Call(callArgs)

			ret := convertFromGoValue(callResult[0])

			return []wasmerGo.Value{ret}, nil
		},
	)

	w.importObject.Register(namespace, map[string]wasmerGo.IntoExtern{
		funcName: fwasmer,
	})

	return nil
}

func (w *Instance) Malloc(size int32) (uint64, error) {
	if !w.checkStart() {
		log.DefaultLogger.Errorf("[wasmer][instance] call malloc before starting instance")
		return 0, ErrInstanceNotStart
	}

	malloc, err := w.GetExportsFunc("malloc")
	if err != nil {
		return 0, err
	}

	addr, err := malloc.Call(size)
	if err != nil {
		w.HandleError(err)
		return 0, err
	}

	return uint64(addr.(int32)), nil
}

func (w *Instance) GetExportsFunc(funcName string) (types.WasmFunction, error) {
	if !w.checkStart() {
		log.DefaultLogger.Errorf("[wasmer][instance] call GetExportsFunc before starting instance")
		return nil, ErrInstanceNotStart
	}

	if v, ok := w.funcCache.Load(funcName); ok {
		return v.(*wasmerGo.Function), nil
	}

	f, err := w.instance.Exports.GetRawFunction(funcName)
	if err != nil {
		return nil, err
	}

	w.funcCache.Store(funcName, f)

	return f, nil
}

func (w *Instance) GetExportsMem(memName string) ([]byte, error) {
	if !w.checkStart() {
		log.DefaultLogger.Errorf("[wasmer][instance] call GetExportsMem before starting instance")
		return nil, ErrInstanceNotStart
	}

	if w.memory == nil {
		m, err := w.instance.Exports.GetMemory(memName)
		if err != nil {
			return nil, err
		}

		w.memory = m
	}

	return w.memory.Data(), nil
}

func (w *Instance) GetMemory(addr uint64, size uint64) ([]byte, error) {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return nil, err
	}

	if int(addr) > len(mem) || int(addr+size) > len(mem) {
		return nil, ErrAddrOverflow
	}

	return mem[addr : addr+size], nil
}

func (w *Instance) PutMemory(addr uint64, size uint64, content []byte) error {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return err
	}

	if int(addr) > len(mem) || int(addr+size) > len(mem) {
		return ErrAddrOverflow
	}

	copySize := uint64(len(content))
	if size < copySize {
		copySize = size
	}

	copy(mem[addr:], content[:copySize])

	return nil
}

func (w *Instance) GetByte(addr uint64) (byte, error) {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return 0, err
	}

	if int(addr) > len(mem) {
		return 0, ErrAddrOverflow
	}

	return mem[addr], nil
}

func (w *Instance) PutByte(addr uint64, b byte) error {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return err
	}

	if int(addr) > len(mem) {
		return ErrAddrOverflow
	}

	mem[addr] = b

	return nil
}

func (w *Instance) GetUint32(addr uint64) (uint32, error) {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return 0, err
	}

	if int(addr) > len(mem) || int(addr+4) > len(mem) {
		return 0, ErrAddrOverflow
	}

	return binary.LittleEndian.Uint32(mem[addr:]), nil
}

func (w *Instance) PutUint32(addr uint64, value uint32) error {
	mem, err := w.GetExportsMem("memory")
	if err != nil {
		return err
	}

	if int(addr) > len(mem) || int(addr+4) > len(mem) {
		return ErrAddrOverflow
	}

	binary.LittleEndian.PutUint32(mem[addr:], value)

	return nil
}

func (w *Instance) HandleError(err error) {
	var trapError *wasmerGo.TrapError
	if !errors.As(err, &trapError) {
		return
	}

	trace := trapError.Trace()
	if trace == nil {
		return
	}

	log.DefaultLogger.Errorf("[wasmer][instance] HandleError err: %v, trace:", err)

	if w.debug == nil {
		// do not have dwarf debug info
		for _, t := range trace {
			log.DefaultLogger.Errorf("[wasmer][instance]\t funcIndex: %v, funcOffset: 0x%08x, moduleOffset: 0x%08x",
				t.FunctionIndex(), t.FunctionOffset(), t.ModuleOffset())
		}
	} else {
		for _, t := range trace {
			pc := uint64(t.ModuleOffset())
			line := w.debug.SeekPC(pc)
			if line != nil {
				log.DefaultLogger.Errorf("[wasmer][instance]\t funcIndex: %v, funcOffset: 0x%08x, pc: 0x%08x %v:%v",
					t.FunctionIndex(), t.FunctionOffset(), pc, line.File.Name, line.Line)
			} else {
				log.DefaultLogger.Errorf("[wasmer][instance]\t funcIndex: %v, funcOffset: 0x%08x, pc: 0x%08x fail to seek pc",
					t.FunctionIndex(), t.FunctionOffset(), t.ModuleOffset())
			}
		}
	}
}
