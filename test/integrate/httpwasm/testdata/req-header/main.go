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

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

// build main like below:
//
//	tinygo build -o main.wasm -scheduler=none --no-debug -target=wasi main.go
func main() {
	handler.HandleRequestFn = setHeader
}

// guest can define the context ID, or it can be looked up in host config.
var contextID int

// setHeader has similar logic to proxywasm for fair comparisons.
func setHeader(req api.Request, resp api.Response) (next bool, reqCtx uint32) {
	contextID++
	req.Headers().Set("Wasm-Context", strconv.Itoa(contextID))
	next = true
	return
}
