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

package proxywasm010

import (
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

func init() {
	abi.RegisterABI(proxywasm.ProxyWasmABI_0_1_0, ABIContextFactory)
}

func ABIContextFactory(instance types.WasmInstance) types.ABI {
	return &ABIContext{
		proxywasm.ABIContext{
			Imports:  &DefaultImportsHandler{Instance: instance},
			Instance: instance,
		},
	}
}

// ABIContext is a wrapper for proxywasm-go-host/proxywasm.ABIContext
// implement types.ABI
type ABIContext struct {
	proxywasm.ABIContext
}

// implement types.ABI
func (ctx *ABIContext) GetABIImports() interface{} {
	return ctx.ABIContext.GetImports()
}

func (ctx *ABIContext) SetABIImports(imports interface{}) {
	if v, ok := imports.(proxywasm.ImportsHandler); ok {
		ctx.ABIContext.SetImports(v)
	}
}

func (ctx *ABIContext) GetABIExports() interface{} {
	return ctx.ABIContext.GetExports()
}

// implement types.ABIHandler
func (ctx *ABIContext) OnInstanceCreate(instance types.WasmInstance) {
	if err := instance.RegisterImports(ctx.Name()); err != nil {
		panic(err)
	}
}

func (ctx *ABIContext) OnInstanceStart(instance types.WasmInstance) {}

func (ctx *ABIContext) OnInstanceDestroy(instance types.WasmInstance) {}
