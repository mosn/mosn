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

package moe

import (
	"context"
	"runtime"
	"strconv"

	udpa "github.com/cncf/xds/go/udpa/type/v1"
	"google.golang.org/protobuf/types/known/anypb"
	mosnApi "mosn.io/api"
	"mosn.io/envoy-go-extension/pkg/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/streamfilter"
	mosnSync "mosn.io/mosn/pkg/sync"
	"mosn.io/mosn/pkg/types"
)

var workerPool mosnSync.WorkerPool

func init() {
	poolSize := runtime.NumCPU() * 256
	workerPool = mosnSync.NewWorkerPool(poolSize)
}

func ConfigFactory(config interface{}) api.HttpFilterFactory {
	ap, ok := config.(*anypb.Any)
	if !ok {
		return nil
	}
	configStruct := &udpa.TypedStruct{}
	if err := ap.UnmarshalTo(configStruct); err != nil {
		log.DefaultLogger.Errorf("[moe] ConfigFactory fail to unmarshal, err: %v", err)
		return nil
	}
	v, ok := configStruct.Value.AsMap()["filter_chain"]
	if !ok {
		log.DefaultLogger.Errorf("[moe] ConfigFactory fail to get filter_chain")
		return nil
	}
	filterChainName := v.(string)

	return func(callbacks api.FilterCallbackHandler) api.HttpFilter {
		ctx := buffer.NewBufferPoolContext(context.Background())
		ctx = variable.NewVariableContext(ctx)

		buf := streamBufferByContext(ctx)
		buf.stream.ctx = ctx
		buf.stream.filterChainName = filterChainName
		buf.stream.filterChain = CreateStreamFilterChain(ctx, filterChainName)
		buf.stream.workPool = workerPool
		buf.stream.callbacks = callbacks

		buf.stream.filterChain.SetReceiveFilterHandler(&buf.stream)
		buf.stream.filterChain.SetSenderFilterHandler(&buf.stream)

		return &buf.stream
	}
}

type ActiveStream struct {
	ctx                 context.Context
	filterChainName     string
	filterChain         *streamfilter.DefaultStreamFilterChainImpl
	currentReceivePhase mosnApi.ReceiverFilterPhase
	callbacks           api.FilterCallbackHandler
	reqHeader           mosnApi.HeaderMap
	reqBody             mosnApi.IoBuffer
	reqTrailer          mosnApi.HeaderMap
	respHeader          mosnApi.HeaderMap
	respBody            mosnApi.IoBuffer
	respTrailer         mosnApi.HeaderMap
	workPool            mosnSync.WorkerPool
	hijack              bool
}

var _ api.HttpFilter = (*ActiveStream)(nil)
var _ mosnApi.StreamReceiverFilterHandler = (*ActiveStream)(nil)
var _ mosnApi.StreamSenderFilterHandler = (*ActiveStream)(nil)

func (s *ActiveStream) SetCurrentReceiverPhase(phase mosnApi.ReceiverFilterPhase) {
	s.currentReceivePhase = phase
}

func (s *ActiveStream) runReceiverFilters() {
	s.workPool.ScheduleAuto(func() {
		s.SetCurrentReceiverPhase(mosnApi.BeforeRoute)
		s.filterChain.RunReceiverFilter(s.ctx, mosnApi.BeforeRoute, s.reqHeader, s.reqBody, s.reqTrailer, nil)
		if !s.hijack {
			s.callbacks.Continue(api.Continue)
		}
	})
}

func (s *ActiveStream) runSenderFilters() {
	s.workPool.ScheduleAuto(func() {
		s.filterChain.RunSenderFilter(s.ctx, mosnApi.BeforeSend, s.respHeader, s.respBody, s.respTrailer, nil)
		if !s.hijack {
			s.callbacks.Continue(api.Continue)
		}
	})
}

func (s *ActiveStream) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	s.reqHeader = &headerMapImpl{header}

	_ = variable.Set(s.ctx, types.VariableDownStreamProtocol, protocol.HTTP1)
	_ = variable.SetString(s.ctx, types.VarHost, header.Host())
	_ = variable.SetString(s.ctx, types.VarIstioHeaderHost, header.Host())
	_ = variable.SetString(s.ctx, types.VarMethod, header.Method())
	_ = variable.SetString(s.ctx, types.VarPath, header.Path())

	if endStream {
		s.runReceiverFilters()
		return api.Running
	}

	return api.StopAndBuffer
}

func (s *ActiveStream) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	s.reqBody = &bufferImpl{buffer}
	if endStream {
		s.runReceiverFilters()
		return api.Running
	}
	return api.StopAndBuffer
}

func (s *ActiveStream) DecodeTrailers(trailer api.RequestTrailerMap) api.StatusType {
	s.reqTrailer = &headerMapImpl{trailer}
	s.runReceiverFilters()

	return api.Running
}

func (s *ActiveStream) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	s.respHeader = &headerMapImpl{header}

	_ = variable.SetString(s.ctx, types.VarHeaderStatus, strconv.Itoa(header.Status()))

	if endStream {
		s.runSenderFilters()
		return api.Running
	}

	return api.StopAndBuffer
}

func (s *ActiveStream) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	s.respBody = &bufferImpl{buffer}
	if endStream {
		s.runSenderFilters()
		return api.Running
	}

	return api.StopAndBuffer
}

func (s *ActiveStream) EncodeTrailers(trailer api.ResponseTrailerMap) api.StatusType {
	s.respTrailer = &headerMapImpl{trailer}
	s.runSenderFilters()
	return api.Running
}

func (s *ActiveStream) OnDestroy(reason api.DestroyReason) {
	s.filterChain.OnDestroy()
	DestroyStreamFilterChain(s.filterChain)

	if ctx := buffer.PoolContext(s.ctx); ctx != nil {
		ctx.Give()
	}
}

func (s *ActiveStream) AppendHeaders(headers mosnApi.HeaderMap, endStream bool) {
	s.respHeader = headers
	if endStream {
		s.SendDirectResponse(s.respHeader, s.respBody, s.respTrailer)
	}
}

func (s *ActiveStream) AppendData(buf mosnApi.IoBuffer, endStream bool) {
	s.respBody = buf
	if endStream {
		s.SendDirectResponse(s.respHeader, s.respBody, s.respTrailer)
	}
}

func (s *ActiveStream) AppendTrailers(trailers mosnApi.HeaderMap) {
	s.respTrailer = trailers
	s.SendDirectResponse(s.respHeader, s.respBody, s.respTrailer)
}

func (s *ActiveStream) SendHijackReply(code int, headers mosnApi.HeaderMap) {
	h := make(map[string]string)
	headers.Range(func(key, value string) bool {
		h[key] = value
		return true
	})

	s.hijack = true
	s.callbacks.SendLocalReply(code, "", h, 0, "")
}

func (s *ActiveStream) SendHijackReplyWithBody(code int, headers mosnApi.HeaderMap, body string) {
	h := make(map[string]string)
	headers.Range(func(key, value string) bool {
		h[key] = value
		return true
	})

	s.hijack = true
	s.callbacks.SendLocalReply(code, body, h, 0, "")
}

func (s *ActiveStream) SendDirectResponse(headers mosnApi.HeaderMap, buf mosnApi.IoBuffer, trailers mosnApi.HeaderMap) {
	codeStr, err := variable.GetString(s.ctx, types.VarHeaderStatus)
	if err != nil {
		return
	}

	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return
	}

	h := make(map[string]string)
	headers.Range(func(key, value string) bool {
		h[key] = value
		return true
	})

	s.callbacks.SendLocalReply(code, buf.String(), h, 0, "")
}

func (s *ActiveStream) TerminateStream(code int) bool {
	if s.respHeader != nil {
		return false
	}

	h := make(map[string]string)
	s.reqHeader.Range(func(key, value string) bool {
		h[key] = value
		return true
	})

	s.hijack = true
	s.callbacks.SendLocalReply(code, "", h, 0, "")

	return true
}

func (s *ActiveStream) GetRequestHeaders() mosnApi.HeaderMap {
	return s.reqHeader
}

func (s *ActiveStream) SetRequestHeaders(headers mosnApi.HeaderMap) {
	panic("implement me")
}

func (s *ActiveStream) GetRequestData() mosnApi.IoBuffer {
	return s.reqBody
}

func (s *ActiveStream) SetRequestData(buf mosnApi.IoBuffer) {
	panic("implement me")
}

func (s *ActiveStream) GetRequestTrailers() mosnApi.HeaderMap {
	return s.reqTrailer
}

func (s *ActiveStream) SetRequestTrailers(trailers mosnApi.HeaderMap) {
	panic("implement me")
}

func (s *ActiveStream) GetFilterCurrentPhase() mosnApi.ReceiverFilterPhase {
	return s.currentReceivePhase
}

func (s *ActiveStream) Route() mosnApi.Route {
	panic("implement me")
}

func (s *ActiveStream) RequestInfo() mosnApi.RequestInfo {
	panic("implement me")
}

func (s *ActiveStream) Connection() mosnApi.Connection {
	panic("implement me")
}

func (s *ActiveStream) GetResponseHeaders() mosnApi.HeaderMap {
	return s.respHeader
}

func (s *ActiveStream) SetResponseHeaders(headers mosnApi.HeaderMap) {
	panic("implement me")
}

func (s *ActiveStream) GetResponseData() mosnApi.IoBuffer {
	return s.respBody
}

func (s *ActiveStream) SetResponseData(buf mosnApi.IoBuffer) {
	panic("implement me")
}

func (s *ActiveStream) GetResponseTrailers() mosnApi.HeaderMap {
	return s.respTrailer
}

func (s *ActiveStream) SetResponseTrailers(trailers mosnApi.HeaderMap) {
	panic("implement me")
}
