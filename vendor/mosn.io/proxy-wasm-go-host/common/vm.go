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

package common

// WasmVM represents the wasm vm(engine)
type WasmVM interface {
	// Name returns the name of wasm vm(engine)
	Name() string

	// Init got called when creating a new wasm vm(engine)
	Init()

	// NewModule compiles the 'wasmBytes' into a wasm module
	NewModule(wasmBytes []byte) WasmModule
}

// WasmModule represents the wasm module
type WasmModule interface {
	// Init got called when creating a new wasm module
	Init()

	// NewInstance instantiates and returns a new wasm instance
	NewInstance() WasmInstance

	// GetABINameList returns the abi name list exported by wasm module
	GetABINameList() []string
}

// WasmInstance represents the wasm instance
type WasmInstance interface {
	// Start starts the wasm instance
	Start() error

	// Stop stops the wasm instance
	Stop()

	// RegisterFunc registers a func to the wasm instance, should be called before Start()
	RegisterFunc(namespace string, funcName string, f interface{}) error

	// GetExportsFunc returns the exported func of the wasm instance
	GetExportsFunc(funcName string) (WasmFunction, error)

	// GetExportsMem returns the exported mem of the wasm instance
	GetExportsMem(memName string) ([]byte, error)

	// GetMemory returns wasm mem bytes from specified addr and size
	GetMemory(addr uint64, size uint64) ([]byte, error)

	// PutMemory sets wasm mem bytes to specified addr and size
	PutMemory(addr uint64, size uint64, content []byte) error

	// GetByte returns one wasm byte from specified addr
	GetByte(addr uint64) (byte, error)

	// PutByte sets one wasms bytes to specified addr
	PutByte(addr uint64, b byte) error

	// GetUint32 returns uint32 from specified addr
	GetUint32(addr uint64) (uint32, error)

	// PutUint32 set uint32 to specified addr
	PutUint32(addr uint64, value uint32) error

	// Malloc allocates size of mem from wasm default memory
	Malloc(size int32) (uint64, error)

	// GetData returns user-defined data
	GetData() interface{}

	// SetData sets user-defined data into the wasm instance
	SetData(data interface{})

	// Acquire increases the ref count of the wasm instance
	Acquire() bool

	// Release decreases the ref count of the wasm instance
	Release()

	// Lock gets the exclusive ownership of the wasm instance
	// and sets the user-defined data
	Lock(data interface{})

	// Unlock releases the exclusive ownership of the wasm instance
	// and sets the users-defined data to nil
	Unlock()

	// GetModule returns the wasm module of current instance
	GetModule() WasmModule

	// HandlerError processes the encountered err
	HandleError(err error)
}

// WasmFunction is the func exported by wasm module
type WasmFunction interface {
	// Call invokes the wasm func
	Call(args ...interface{}) (interface{}, error)
}
