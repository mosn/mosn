// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxywasm

import "github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"

//export proxy_on_new_connection
func proxyOnNewConnection(contextID uint32) types.Action {
	ctx, ok := currentState.streams[contextID]
	if !ok {
		panic("invalid context")
	}
	currentState.setActiveContextID(contextID)
	return ctx.OnNewConnection()
}

//export proxy_on_downstream_data
func proxyOnDownstreamData(contextID uint32, dataSize int, endOfStream bool) types.Action {
	ctx, ok := currentState.streams[contextID]
	if !ok {
		panic("invalid context")
	}
	currentState.setActiveContextID(contextID)
	return ctx.OnDownstreamData(dataSize, endOfStream)
}

//export proxy_on_downstream_connection_close
func proxyOnDownstreamConnectionClose(contextID uint32, pType types.PeerType) {
	ctx, ok := currentState.streams[contextID]
	if !ok {
		panic("invalid context")
	}
	currentState.setActiveContextID(contextID)
	ctx.OnDownstreamClose(pType)
}

//export proxy_on_upstream_data
func proxyOnUpstreamData(contextID uint32, dataSize int, endOfStream bool) types.Action {
	ctx, ok := currentState.streams[contextID]
	if !ok {
		panic("invalid context")
	}
	currentState.setActiveContextID(contextID)
	return ctx.OnUpstreamData(dataSize, endOfStream)
}

//export proxy_on_upstream_connection_close
func proxyOnUpstreamConnectionClose(contextID uint32, pType types.PeerType) {
	ctx, ok := currentState.streams[contextID]
	if !ok {
		panic("invalid context")
	}
	currentState.setActiveContextID(contextID)
	ctx.OnUpstreamClose(pType)
}
