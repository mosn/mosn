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

package proxywasm_0_1_0

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/wasm/runtime/wasmer"
)

func TestCallExports(t *testing.T) {
	vm := wasmer.NewWasmerVM()
	module := vm.NewModule([]byte(`
			(module
				(func (export "_start"))
				(func (export "proxy_on_vm_start") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_validate_configuration") (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_configure") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_tick") (param i32) (result))
				(func (export "proxy_on_context_create") (param i32) (param i32) (result))
				(func (export "proxy_on_new_connection") (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_downstream_data") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_upstream_data") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_downstream_connection_close") (param i32) (param i32) (result))
				(func (export "proxy_on_upstream_connection_close") (param i32) (param i32) (result))
				(func (export "proxy_on_request_headers") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_request_metadata") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_request_body") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_request_trailers") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_response_headers") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_response_metadata") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_response_body") (param i32) (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_response_trailers") (param i32) (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_done") (param i32) (result i32) i32.const 0)
				(func (export "proxy_on_log") (param i32) (result))
				(func (export "proxy_on_delete") (param i32) (result))
				(func (export "proxy_on_http_call_response") (param i32) (param i32) (param i32) (param i32) (param i32) (result))
				(func (export "proxy_on_queue_ready") (param i32) (param i32) (result))
				(func (export "proxy_on_foreign_function") (param i32) (result i32) i32.const 0)
			)
	`))
	ins := module.NewInstance()
	assert.Nil(t, ins.Start())

	abi := abiContextFactory(ins)
	exports := abi.GetExports().(Exports)

	var err error

	assert.Nil(t, exports.ProxyOnContextCreate(0, 0))

	_, err = exports.ProxyOnDone(0)
	assert.Nil(t, err)

	assert.Nil(t, exports.ProxyOnLog(0))
	assert.Nil(t, exports.ProxyOnDelete(0))

	_, err = exports.ProxyOnVmStart(0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnConfigure(0, 0)
	assert.Nil(t, err)

	assert.Nil(t, exports.ProxyOnTick(0))

	_, err = exports.ProxyOnNewConnection(0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnDownstreamData(0, 0, 0)
	assert.Nil(t, err)

	assert.Nil(t, exports.ProxyOnDownstreamConnectionClose(0, 0))

	_, err = exports.ProxyOnUpstreamData(0, 0, 0)
	assert.Nil(t, err)

	assert.Nil(t, exports.ProxyOnUpstreamConnectionClose(0, 0))

	_, err = exports.ProxyOnRequestHeaders(0, 0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnRequestBody(0, 0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnRequestTrailers(0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnRequestMetadata(0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnResponseHeaders(0, 0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnResponseBody(0, 0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnResponseTrailers(0, 0)
	assert.Nil(t, err)

	_, err = exports.ProxyOnResponseMetadata(0, 0)
	assert.Nil(t, err)

	assert.Nil(t, exports.ProxyOnHttpCallResponse(0, 0, 0, 0, 0))

	assert.Nil(t, exports.ProxyOnQueueReady(0, 0))
}
