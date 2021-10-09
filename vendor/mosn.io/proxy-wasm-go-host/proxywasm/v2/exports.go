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

package v2

func (a *ABIContext) CallWasmFunction(funcName string, args ...interface{}) (interface{}, Action, error) {
	ff, err := a.Instance.GetExportsFunc(funcName)
	if err != nil {
		return nil, ActionContinue, err
	}

	res, err := ff.Call(args...)
	if err != nil {
		a.Instance.HandleError(err)
		return nil, ActionContinue, err
	}

	// if we have sync call, e.g. HttpCall, then unlock the wasm instance and wait until it resp
	action := a.Imports.Wait()

	return res, action, nil
}

func (a *ABIContext) ProxyOnMemoryAllocate(memorySize int32) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_memory_allocate", memorySize)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func (a *ABIContext) ProxyOnContextCreate(contextID int32, parentContextID int32, contextType ContextType) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_context_create", contextID, parentContextID, contextType)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func (a *ABIContext) ProxyOnContextFinalize(contextID int32) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_context_finalize", contextID)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func (a *ABIContext) ProxyOnVmStart(vmID int32, vmConfigurationSize int32) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_vm_start", vmID, vmConfigurationSize)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func (a *ABIContext) ProxyOnPluginStart(pluginID int32, pluginConfigurationSize int32) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_plugin_start", pluginID, pluginConfigurationSize)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func (a *ABIContext) ProxyOnNewConnection(streamID int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_new_connection", streamID)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnDownstreamData(streamID int32, dataSize int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_downstream_data", streamID, dataSize, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnDownstreamClose(contextID int32, closeSource CloseSourceType) error {
	_, _, err := a.CallWasmFunction("proxy_on_downstream_close", contextID, closeSource)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnUpstreamData(streamID int32, dataSize int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_upstream_data", streamID, dataSize, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnUpstreamClose(streamID int32, closeSource CloseSourceType) error {
	_, _, err := a.CallWasmFunction("proxy_on_upstream_close", streamID, closeSource)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnHttpRequestHeaders(streamID int32, numHeaders int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_request_headers", streamID, numHeaders, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpRequestBody(streamID int32, bodySize int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_request_body", streamID, bodySize, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpRequestTrailers(streamID int32, numTrailers int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_request_trailers", streamID, numTrailers, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpRequestMetadata(streamID int32, numElements int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_request_metadata", streamID, numElements)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpResponseHeaders(streamID int32, numHeaders int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_response_headers", streamID, numHeaders, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpResponseBody(streamID int32, bodySize int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_response_body", streamID, bodySize, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpResponseTrailers(streamID int32, numTrailers int32, endOfStream int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_response_trailers", streamID, numTrailers, endOfStream)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnHttpResponseMetadata(streamID int32, numElements int32) (Action, error) {
	_, action, err := a.CallWasmFunction("proxy_on_http_response_metadata", streamID, numElements)
	if err != nil {
		return ActionPause, err
	}
	return action, nil
}

func (a *ABIContext) ProxyOnQueueReady(queueID int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_queue_ready", queueID)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnTimerReady(timerID int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_timer_ready", timerID)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnHttpCallResponse(calloutID int32, numHeaders int32, bodySize int32, numTrailers int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_http_call_response", calloutID, numHeaders, bodySize, numTrailers)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseHeaderMetadata(calloutID int32, numHeaders int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_grpc_call_response_header_metadata", calloutID, numHeaders)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseMessage(calloutID int32, messageSize int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_grpc_call_response_message", calloutID, messageSize)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseTrailerMetadata(calloutID int32, numTrailers int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_grpc_call_response_trailer_metadata", calloutID, numTrailers)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnGrpcCallClose(calloutID int32, statusCode int32) error {
	_, _, err := a.CallWasmFunction("proxy_on_grpc_call_close", calloutID, statusCode)
	if err != nil {
		return err
	}
	return nil
}

func (a *ABIContext) ProxyOnCustomCallback(customCallbackID int32, parametersSize int32) (int32, error) {
	res, _, err := a.CallWasmFunction("proxy_on_custom_callback", customCallbackID, parametersSize)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}
