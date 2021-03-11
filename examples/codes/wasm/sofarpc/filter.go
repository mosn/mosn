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
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

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

// override
func (ctx *myHttpContext) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	// get all header
	proxywasm.LogInfo("proxywasm get request headers...")
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
	}
	for _, h := range hs {
		proxywasm.LogInfof("request header from go wasm --> %s: %s", h[0], h[1])
	}

	return types.ActionContinue
}

// override
func (ctx *myHttpContext) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	// get request body
	proxywasm.LogInfo("proxywasm get request body...")
	bytes, err := proxywasm.GetHttpRequestBody(0, 100)
	if err != nil {
		proxywasm.LogCriticalf("failed to get request body: %v", err)
	}
	proxywasm.LogInfof("proxywasm.GetHttpRequestBody [%s]", string(bytes))

	return types.ActionContinue
}
