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

const calloutURL string = "http://127.0.0.1:2046/"

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
	// http callout with the original headers
	_, err := proxywasm.DispatchHttpCall(calloutURL, [][2]string{}, "", [][2]string{}, 50000, ctx.httpCallResponseCallback)
	if err != nil {
		proxywasm.LogErrorf("dispatch http call failed: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func (ctx *myHttpContext) httpCallResponseCallback(numHeaders, bodySize, numTrailers int) {
	// get response headers
	hs, err := proxywasm.GetHttpCallResponseHeaders()
	if err != nil {
		proxywasm.LogErrorf("failed to get response headers: %v", err)
		return
	}
	for _, h := range hs {
		proxywasm.LogInfof("response header from %s: %s: %s", calloutURL, h[0], h[1])
	}

	// get response body
	b, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
	if err != nil {
		proxywasm.LogErrorf("failed to get response body: %v", err)
		return
	}

	proxywasm.LogInfof("response body from %s: %v", calloutURL, string(b))
}
