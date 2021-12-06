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
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

const (
	ResponseType              = 0
	RequestType               = 1
	RequestOneWayType         = 2
	HeartBeatFlag        byte = 1 << 5
	RpcHeaderLength           = 16
	RpcMagic                  = 0xAF
	RpcVersion                = 1
	RpcRequestFlag       byte = 1 << 6
	RpcOneWayRequestFlag byte = 1<<6 | 1<<7
	RpcResponseFlag      byte = 0
	RpcIdIndex                = 4
	RpcTimeout                = "timeout"
	RpcResponseStatus         = 0
	UnKnownMagicType          = "unknown magic type"
	UnKnownProtocolType       = "unknown protocol code type"
	UnKnownRpcFlagType        = "unknown protocol flag type"
)

type ProtocolConfig struct {
	VmConfig    *v2.WasmVmConfig `json:"vm_config,omitempty"`
	InstanceNum int              `json:"instance_num,omitempty"`
	// protocol feature field
	poolMode api.PoolMode

	SubProtocol       string `json:"protocol,omitempty"`
	FromWasmPlugin    string `json:"from_wasm_plugin,omitempty"`
	RootContextID     int32  `json:"root_id,omitempty"`
	PoolMode          string `json:"pool_mode,omitempty"`
	DisableWorkerPool bool   `json:"disable_worker_pool,omitempty"`
	PluginGenerateID  bool   `json:"plugin_generate_id,omitempty"`
}

// extension for protocol
const (
	BufferTypeDecodeData proxywasm.BufferType = 13
	BufferTypeEncodeData proxywasm.BufferType = 14
	StatusNeedMoreData   proxywasm.WasmResult = 99
)
