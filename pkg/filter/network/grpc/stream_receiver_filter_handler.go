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

package grpc

import (
	"errors"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type grpcStreamReceiverFilterHandler struct {
	sfc *grpcStreamFilterChain
}

func newStreamReceiverFilterHandler(sfc *grpcStreamFilterChain) *grpcStreamReceiverFilterHandler {
	f := &grpcStreamReceiverFilterHandler{sfc}
	return f
}

func (h grpcStreamReceiverFilterHandler) Route() api.Route {
	log.DefaultLogger.Warnf("Route() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) RequestInfo() api.RequestInfo {
	log.DefaultLogger.Warnf("RequestInfo() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) Connection() api.Connection {
	log.DefaultLogger.Warnf("Connection() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) AppendHeaders(headers api.HeaderMap, endStream bool) {
	log.DefaultLogger.Warnf("AppendHeaders() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) AppendData(buf api.IoBuffer, endStream bool) {
	log.DefaultLogger.Warnf("AppendData() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) AppendTrailers(trailers api.HeaderMap) {
	log.DefaultLogger.Warnf("AppendTrailers() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) SendHijackReply(code int, headers api.HeaderMap) {
	log.DefaultLogger.Warnf("SendHijackReply() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) SendHijackReplyWithBody(code int, headers api.HeaderMap, body string) {
	log.DefaultLogger.Warnf("SendHijackReplyWithBody() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) SendDirectResponse(headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) {
	h.sfc.err = errors.New(buf.String())
}

func (h grpcStreamReceiverFilterHandler) TerminateStream(code int) bool {
	log.DefaultLogger.Warnf("TerminateStream() not implemented yet")
	return false
}

func (h grpcStreamReceiverFilterHandler) GetRequestHeaders() api.HeaderMap {
	log.DefaultLogger.Warnf("GetRequestHeaders() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) SetRequestHeaders(headers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetRequestHeaders() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) GetRequestData() api.IoBuffer {
	log.DefaultLogger.Warnf("GetRequestData() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) SetRequestData(buf api.IoBuffer) {
	log.DefaultLogger.Warnf("SetRequestData() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) GetRequestTrailers() api.HeaderMap {
	log.DefaultLogger.Warnf("GetRequestTrailers() not implemented yet")
	return nil
}

func (h grpcStreamReceiverFilterHandler) SetRequestTrailers(trailers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetRequestTrailers() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) SetConvert(on bool) {
	log.DefaultLogger.Warnf("SetConvert() not implemented yet")
}

func (h grpcStreamReceiverFilterHandler) GetFilterCurrentPhase() api.ReceiverFilterPhase {
	// default AfterRoute
	p := api.AfterRoute

	switch h.sfc.phase {
	case types.DownFilterAfterRoute:
		p = api.AfterRoute
	}
	return p
}
