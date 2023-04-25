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

package wasm

import (
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// var vmMap = make(map[string]types.WasmVM)
var vmRegistry = sync.Map{}

// RegisterWasmEngine registers a wasm vm(engine).
func RegisterWasmEngine(name string, engine types.WasmVM) {
	if name == "" || engine == nil {
		log.DefaultLogger.Errorf("[wasm][vm] RegisterWasmEngine invalid param, engine: %v", name)
		return
	}

	log.DefaultLogger.Infof("[wasm][vm] RegisterWasmEngine engine: %v", name)
	vmRegistry.Store(name, engine)
}

// GetWasmEngine returns the wasm vm(engine) by name.
func GetWasmEngine(name string) types.WasmVM {
	if v, ok := vmRegistry.Load(name); ok {
		if engine, ok := v.(types.WasmVM); ok {
			return engine
		}
	}

	log.DefaultLogger.Errorf("[wasm][vm] GetWasmEngine not found, engine: %v", name)
	return nil
}
