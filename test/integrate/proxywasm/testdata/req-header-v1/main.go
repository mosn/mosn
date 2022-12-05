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

package main

import (
	"strconv"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

// build main like below with proxy-wasm-go-sdk v0.0.14 which works with
// TinyGo 0.19.0 (which works with Go 1.16).
//
//	tinygo build -o main.wasm -scheduler=none -target=wasi --no-debug -wasm-abi=generic -tags 'abi_010' ./main.go
//
// Note: Old tooling is needed because mosn.io/proxy-wasm-go-host v2 is
// incompatible with current proxy-wasm SDKs.
func main() {
	proxywasm.SetNewHttpContext(newHttpContext)
}

type myHttpContext struct {
	// you must embed the default context so that you need not to re-implement all the methods by yourself
	proxywasm.DefaultHttpContext
	contextID uint32
}

func newHttpContext(rootContextID, contextID uint32) proxywasm.HttpContext {
	return &myHttpContext{contextID: contextID}
}

func (ctx *myHttpContext) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	key := "Wasm-Context"
	err := proxywasm.SetHttpRequestHeader(key, strconv.Itoa(int(ctx.contextID)))
	if err != nil {
		proxywasm.LogCritical("failed to set request header: " + key)
	}
	return types.ActionContinue
}
