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
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

type grpcStreamSenderFilterHandler struct {
	sfc *grpcStreamFilterChain
}

func newStreamSenderFilterHandler(sfc *grpcStreamFilterChain) *grpcStreamSenderFilterHandler {
	f := &grpcStreamSenderFilterHandler{sfc}
	return f
}

func (g grpcStreamSenderFilterHandler) Route() api.Route {
	log.DefaultLogger.Warnf("Route() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) RequestInfo() api.RequestInfo {
	log.DefaultLogger.Warnf("RequestInfo() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) Connection() api.Connection {
	log.DefaultLogger.Warnf("Connection() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) GetResponseHeaders() api.HeaderMap {
	log.DefaultLogger.Warnf("GetResponseHeaders() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseHeaders(headers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetResponseHeaders() not implemented yet")
}

func (g grpcStreamSenderFilterHandler) GetResponseData() api.IoBuffer {
	log.DefaultLogger.Warnf("GetResponseData() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseData(buf api.IoBuffer) {
	log.DefaultLogger.Warnf("SetResponseData() not implemented yet")
}

func (g grpcStreamSenderFilterHandler) GetResponseTrailers() api.HeaderMap {
	log.DefaultLogger.Warnf("GetResponseTrailers() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseTrailers(trailers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetResponseTrailers() not implemented yet")
}
