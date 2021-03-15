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

func (a *ABIContext) ProxyOnContextCreate(contextId int32, parentContextId int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_context_create")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId, parentContextId)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnDone(contextId int32) (int32, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_done")
	if err != nil {
		return 0, err
	}

	res, err := ff.Call(contextId)
	if err != nil {
		a.Instance.HandleError(err)
		return 0, err
	}

	a.Imports.Wait()

	return res.(int32), nil
}

func (a *ABIContext) ProxyOnLog(contextId int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_log")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnVmStart(rootContextId int32, vmConfigurationSize int32) (int32, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_vm_start")
	if err != nil {
		return 0, err
	}

	res, err := ff.Call(rootContextId, vmConfigurationSize)
	if err != nil {
		a.Instance.HandleError(err)
		return 0, err
	}

	a.Imports.Wait()

	return res.(int32), nil
}

func (a *ABIContext) ProxyOnDelete(contextId int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_delete")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnConfigure(rootContextId int32, configurationSize int32) (int32, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_configure")
	if err != nil {
		return 0, err
	}

	res, err := ff.Call(rootContextId, configurationSize)
	if err != nil {
		a.Instance.HandleError(err)
		return 0, err
	}

	a.Imports.Wait()

	return res.(int32), nil
}

func (a *ABIContext) ProxyOnTick(rootContextId int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_tick")
	if err != nil {
		return err
	}

	_, err = ff.Call(rootContextId)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnNewConnection(contextId int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_new_connection")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnDownstreamData(contextId int32, dataLength int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_downstream_data")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, dataLength, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnDownstreamConnectionClose(contextId int32, closeType int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_downstream_connection_close")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId, closeType)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnUpstreamData(contextId int32, dataLength int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_upstream_data")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, dataLength, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnUpstreamConnectionClose(contextId int32, closeType int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_upstream_connection_close")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId, closeType)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnRequestHeaders(contextID int32, numHeaders int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_request_headers")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextID, numHeaders, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnRequestBody(contextId int32, bodyBufferLength int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_request_body")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, bodyBufferLength, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnRequestTrailers(contextId int32, trailers int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_request_trailers")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, trailers)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnRequestMetadata(contextId int32, nElements int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_request_metadata")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, nElements)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnResponseHeaders(contextId int32, headers int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_response_headers")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, headers, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnResponseBody(contextId int32, bodyBufferLength int32, endOfStream int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_response_body")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, bodyBufferLength, endOfStream)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnResponseTrailers(contextId int32, trailers int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_response_trailers")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, trailers)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnResponseMetadata(contextId int32, nElements int32) (Action, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_response_metadata")
	if err != nil {
		return ActionPause, err
	}

	res, err := ff.Call(contextId, nElements)
	if err != nil {
		a.Instance.HandleError(err)
		return ActionPause, err
	}

	a.Imports.Wait()

	return Action(res.(int32)), nil
}

func (a *ABIContext) ProxyOnHttpCallResponse(contextId int32, token int32, headers int32, bodySize int32, trailers int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_http_call_response")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextId, token, headers, bodySize, trailers)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnQueueReady(rootContextId int32, token int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_queue_ready")
	if err != nil {
		return err
	}

	_, err = ff.Call(rootContextId, token)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnMemoryAllocate(size int32) (int32, error) {
	ff, err := a.Instance.GetExportsFunc("proxy_on_memory_allocate")
	if err != nil {
		return 0, err
	}

	res, err := ff.Call(size)
	if err != nil {
		a.Instance.HandleError(err)
		return 0, err
	}

	a.Imports.Wait()

	return res.(int32), nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseHeaderMetadata(contextID int32, calloutID int32, nElements int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_grpc_call_response_header_metadata")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextID, calloutID, nElements)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseMessage(contextID int32, calloutID int32, msgSize int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_grpc_call_response_message")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextID, calloutID, msgSize)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnGrpcCallResponseTrailerMetadata(contextID int32, calloutID int32, nElements int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_grpc_call_response_trailer_metadata")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextID, calloutID, nElements)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}

func (a *ABIContext) ProxyOnGrpcCallClose(contextID int32, calloutID int32, statusCode int32) error {
	ff, err := a.Instance.GetExportsFunc("proxy_on_grpc_call_close")
	if err != nil {
		return err
	}

	_, err = ff.Call(contextID, calloutID, statusCode)
	if err != nil {
		a.Instance.HandleError(err)
		return err
	}

	a.Imports.Wait()

	return nil
}
