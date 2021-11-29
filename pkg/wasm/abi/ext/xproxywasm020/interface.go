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

package xproxywasm020

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

type Exports interface {
	proxywasm.Exports

	// x-protocol
	ProxyDecodeBufferBytes(contextId int32, buf types.IoBuffer) error
	ProxyEncodeRequestBufferBytes(contextId int32, cmd api.XFrame) error
	ProxyEncodeResponseBufferBytes(contextId int32, cmd api.XRespFrame) error

	// x-protocol keepalive
	ProxyKeepAliveBufferBytes(contextId int32, id uint64) error
	ProxyReplyKeepAliveBufferBytes(contextId int32, cmd api.XFrame) error
	ProxyHijackBufferBytes(contextId int32, cmd api.XFrame, statusCode uint32) error
}
