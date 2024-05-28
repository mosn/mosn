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

package proxywasm

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm010"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm020"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	v1Host "mosn.io/proxy-wasm-go-host/proxywasm/v1"
	v2Host "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

// v1 Imports
type v1Imports struct {
	proxywasm010.DefaultImportsHandler
	factory *FilterConfigFactory
	filter  *Filter
}

func (v1 *v1Imports) GetRootContextID() int32 {
	return v1.factory.rootContextID
}

func (v1 *v1Imports) GetVmConfig() common.IoBuffer {
	return v1.factory.GetVmConfig()
}

func (v1 *v1Imports) GetPluginConfig() common.IoBuffer {
	return v1.factory.GetPluginConfig()
}

func (v1 *v1Imports) GetHttpRequestHeader() common.HeaderMap {
	if v1.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: v1.filter.receiverFilterHandler.GetRequestHeaders()}
}

func (v1 *v1Imports) GetHttpRequestBody() common.IoBuffer {
	if v1.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.IoBufferWrapper{IoBuffer: v1.filter.receiverFilterHandler.GetRequestData()}
}

func (v1 *v1Imports) GetHttpRequestTrailer() common.HeaderMap {
	if v1.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: v1.filter.receiverFilterHandler.GetRequestTrailers()}
}

func (v1 *v1Imports) GetHttpResponseHeader() common.HeaderMap {
	if v1.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: v1.filter.senderFilterHandler.GetResponseHeaders()}
}

func (v1 *v1Imports) GetHttpResponseBody() common.IoBuffer {
	if v1.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.IoBufferWrapper{IoBuffer: v1.filter.senderFilterHandler.GetResponseData()}
}

func (v1 *v1Imports) GetHttpResponseTrailer() common.HeaderMap {
	if v1.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm010.HeaderMapWrapper{HeaderMap: v1.filter.senderFilterHandler.GetResponseTrailers()}
}

// v2 Imports
type v2Imports struct {
	proxywasm020.DefaultImportsHandler
	factory *FilterConfigFactory
	filter  *Filter
}

func (v2 *v2Imports) GetRootContextID() int32 {
	return v2.factory.rootContextID
}

func (v2 *v2Imports) GetVmConfig() common.IoBuffer {
	return v2.factory.GetVmConfig()
}

func (v2 *v2Imports) GetPluginConfig() common.IoBuffer {
	return v2.factory.GetPluginConfig()
}

func (v2 *v2Imports) GetHttpRequestHeader() common.HeaderMap {
	if v2.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm020.HeaderMapWrapper{HeaderMap: v2.filter.receiverFilterHandler.GetRequestHeaders()}
}

func (v2 *v2Imports) GetHttpRequestBody() common.IoBuffer {
	if v2.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm020.IoBufferWrapper{IoBuffer: v2.filter.receiverFilterHandler.GetRequestData()}
}

func (v2 *v2Imports) GetHttpRequestTrailer() common.HeaderMap {
	if v2.filter.receiverFilterHandler == nil {
		return nil
	}

	return &proxywasm020.HeaderMapWrapper{HeaderMap: v2.filter.receiverFilterHandler.GetRequestTrailers()}
}

func (v2 *v2Imports) GetHttpResponseHeader() common.HeaderMap {
	if v2.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm020.HeaderMapWrapper{HeaderMap: v2.filter.senderFilterHandler.GetResponseHeaders()}
}

func (v2 *v2Imports) GetHttpResponseBody() common.IoBuffer {
	if v2.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm020.IoBufferWrapper{IoBuffer: v2.filter.senderFilterHandler.GetResponseData()}
}

func (v2 *v2Imports) GetHttpResponseTrailer() common.HeaderMap {
	if v2.filter.senderFilterHandler == nil {
		return nil
	}

	return &proxywasm020.HeaderMapWrapper{HeaderMap: v2.filter.senderFilterHandler.GetResponseTrailers()}
}

// exports
type exportAdapter struct {
	v1 v1Host.Exports
	v2 v2Host.Exports
}

func (e *exportAdapter) ProxyOnContextCreate(contextID int32, parentContextID int32) error {
	if e.v1 != nil {
		return e.v1.ProxyOnContextCreate(contextID, parentContextID)
	} else {
		return e.v2.ProxyOnContextCreate(contextID, parentContextID, v2Host.ContextTypeHttpContext)
	}
}

func (e *exportAdapter) ProxyOnVmStart(rootContextID int32, vmConfigurationSize int32) (int32, error) {
	if e.v1 != nil {
		return e.v1.ProxyOnVmStart(rootContextID, vmConfigurationSize)
	} else {
		return e.v2.ProxyOnVmStart(rootContextID, vmConfigurationSize)
	}
}

func (e *exportAdapter) ProxyOnConfigure(rootContextID int32, pluginConfigurationSize int32) (int32, error) {
	if e.v1 != nil {
		return e.v1.ProxyOnVmStart(rootContextID, pluginConfigurationSize)
	} else {
		return e.v2.ProxyOnVmStart(rootContextID, pluginConfigurationSize)
	}
}

func (e *exportAdapter) ProxyOnDone(contextID int32) (int32, error) {
	if e.v1 != nil {
		return e.v1.ProxyOnDone(contextID)
	} else {
		return e.v2.ProxyOnDone(contextID)
	}
}

func (e *exportAdapter) ProxyOnDelete(contextID int32) error {
	if e.v1 != nil {
		return e.v1.ProxyOnDelete(contextID)
	}
	return nil
}

func (e *exportAdapter) ProxyOnRequestHeaders(contextID int32, headers int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnRequestHeaders(contextID, headers, endOfStream)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnRequestHeaders action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnRequestHeaders(contextID, headers, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnRequestHeaders action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (e *exportAdapter) ProxyOnRequestBody(contextID int32, bodyBufferLength int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnRequestBody(contextID, bodyBufferLength, endOfStream)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnRequestBody action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnRequestBody(contextID, bodyBufferLength, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnRequestBody action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (e *exportAdapter) ProxyOnRequestTrailers(contextID int32, trailers int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnRequestTrailers(contextID, trailers)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnRequestTrailers action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnRequestTrailers(contextID, trailers, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnRequestTrailers action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (e *exportAdapter) ProxyOnResponseHeaders(contextID int32, headers int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnResponseHeaders(contextID, headers, endOfStream)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnResponseHeaders action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnResponseHeaders(contextID, headers, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnResponseHeaders action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (e *exportAdapter) ProxyOnResponseBody(contextID int32, bodyBufferLength int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnResponseBody(contextID, bodyBufferLength, endOfStream)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnRequestBody action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnResponseBody(contextID, bodyBufferLength, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnRequestBody action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (e *exportAdapter) ProxyOnResponseTrailers(contextID int32, trailers int32, endOfStream int32) api.StreamFilterStatus {
	if e.v1 != nil {
		action, err := e.v1.ProxyOnResponseTrailers(contextID, trailers)
		if err != nil || action != v1Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v1] ProxyOnResponseTrailers action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	} else {
		action, err := e.v2.ProxyOnResponseTrailers(contextID, trailers, endOfStream)
		if err != nil || action != v2Host.ActionContinue {
			log.DefaultLogger.Errorf("[proxywasm][filter][v2] ProxyOnResponseTrailers action: %v, err: %v", action, err)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}
