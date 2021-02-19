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
	"strings"

	wasmerGo "github.com/wasmerio/wasmer-go/wasmer"
	"mosn.io/mosn/pkg/types"
)

type Module struct {
	vm            *VM
	module        *wasmerGo.Module
	abiNameList   []string
	moduleImports []moduleImport
}

type moduleImport struct {
	namespace string
	funcName  string
	f         *wasmerGo.Function
}

func NewWasmerModule(vm *VM, module *wasmerGo.Module) *Module {
	m := &Module{
		vm:            vm,
		module:        module,
		moduleImports: make([]moduleImport, 0),
	}

	m.Init()

	return m
}

func (w *Module) Init() {
	w.ensureWASIImports()
	w.abiNameList = w.GetABINameList()
	return
}

func (w *Module) NewInstance() types.WasmInstance {
	return NewWasmerInstance(w.vm, w)
}

func (w *Module) GetABINameList() []string {
	abiNameList := make([]string, 0)

	exportList := w.module.Exports()

	for _, export := range exportList {
		if export.Type().Kind() == wasmerGo.FUNCTION {
			if strings.HasPrefix(export.Name(), "proxy_abi") {
				abiNameList = append(abiNameList, export.Name())
			}
		}
	}

	return abiNameList
}

func (w *Module) ensureWASIImports() {
	importList := w.module.Imports()

	for _, im := range importList {
		if im.Type().Kind() == wasmerGo.FUNCTION && im.Module() == "wasi_unstable" {

			fType := im.Type().IntoFunctionType()

			f := wasmerGo.NewFunction(w.vm.store, wasmerGo.NewFunctionType(fType.Params(), fType.Results()),
				func(values []wasmerGo.Value) ([]wasmerGo.Value, error) {
					return nil, nil
				},
			)

			w.moduleImports = append(w.moduleImports, moduleImport{
				namespace: "wasi_unstable",
				funcName:  im.Name(),
				f:         f,
			})
		}
	}
}
