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

package iot

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"sofastack.io/sofa-mosn/pkg/filter"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/protocol"
	"sofastack.io/sofa-mosn/pkg/protocol/http"
	"sofastack.io/sofa-mosn/pkg/types"
)

const (
	HeaderXCaKey          string = "X-Ca-Key"
	HeaderXCaErrorMessage string = "X-Ca-Error-Message"
)

const (
	ErrorMessageBadRequest          string = "Bad Request."
	ErrorMessageInternalServerError string = "Internal Server Error."
)

const (
	traceLogPath = "/etc/istio/proxy" + string(os.PathSeparator) + "iot_trace.log"
)

type traceLogFilter struct {
	context context.Context

	// request properties
	trace     bool
	source    string
	requestId string
	method    string
	path      string
	request   string
	// response properties
	hasError     bool
	response     string
	responseCode int
	// callbacks
	receiverHandler types.StreamReceiverFilterHandler
	senderHandler   types.StreamSenderFilterHandler
	// logger
	logger *log.Logger
}

type IoTxRequest struct {
	Id      string      `json:"id"`
	Version string      `json:"version"`
	Request interface{} `json:"request"`
	Params  interface{} `json:"params"`
}

type IoTxResponse struct {
	Id           string `json:"id"`
	Code         int32  `json:"code"`
	Message      string `json:"message"`
	LocalizedMsg string `json:"localizedMsg"`
}

func init() {
	filter.RegisterStream("iottracelog", CreateTraceLogFilterFactory)
}

func NewTraceLogFilter(context context.Context) *traceLogFilter {

	var err error
	var logger *log.Logger

	if logger, err = log.GetOrCreateLogger(traceLogPath); err != nil {
		return nil
	}

	return &traceLogFilter{
		context: context,
		logger:  logger,
	}
}

func (f *traceLogFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	xCaKey, ok := headers.Get(HeaderXCaKey)
	f.trace = ok && len(xCaKey) > 0

	if !f.trace {
		return types.StreamFilterContinue
	}

	f.path, _ = headers.Get(protocol.MosnHeaderPathKey)
	f.method, _ = headers.Get(protocol.MosnHeaderMethod)

	f.source = xCaKey

	var request IoTxRequest
	requestStr := buf.String()
	err := json.Unmarshal([]byte(requestStr), &request)

	if err != nil {
		f.hasError = true

		f.receiverHandler.RequestInfo().SetResponseCode(http.BadRequest)

		headers := http.ResponseHeader{}
		headers.Set(HeaderXCaErrorMessage, ErrorMessageBadRequest)
		f.receiverHandler.AppendHeaders(headers, true)
		return types.StreamFilterStop
	}

	f.requestId = request.Id
	f.request = requestStr

	return types.StreamFilterContinue

}

func (f *traceLogFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.receiverHandler = handler
}

func (f *traceLogFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	if !f.trace {
		return types.StreamFilterContinue
	}
	f.responseCode = f.senderHandler.RequestInfo().ResponseCode();
	var response IoTxResponse

	responseStr := buf.String()
	err := json.Unmarshal([]byte(responseStr), &response)
	if err != nil {
		f.hasError = true
		return types.StreamFilterContinue
	}
	json, _ := json.Marshal(response)
	f.response = string(json)

	return types.StreamFilterContinue
}

func (f *traceLogFilter) SetSenderFilterHandler(handler types.StreamSenderFilterHandler) {
	f.senderHandler = handler
}

func (f *traceLogFilter) OnDestroy() {

}

func (f *traceLogFilter) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if !f.trace || reqHeaders == nil || respHeaders == nil || requestInfo == nil {
		return
	}

	f.logger.Printf("source:%s, "+
		"sourceIp:%s, "+
		"path:%s, "+
		"method:%s, "+
		"request:%s, "+
		"statusCode:%d, "+
		"response:%s, "+
		"expireTime:%v",
		f.source,
		requestInfo.DownstreamRemoteAddress().String(),
		f.path,
		f.method,
		f.request,
		f.responseCode,
		f.response,
		requestInfo.Duration().Round(time.Millisecond),
	)
}

type TraceLogFilterConfigFactory struct {
}

func (f *TraceLogFilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewTraceLogFilter(context)
	callbacks.AddStreamReceiverFilter(filter, types.DownFilter)
	callbacks.AddStreamSenderFilter(filter)
	callbacks.AddStreamAccessLog(filter)
}

func CreateTraceLogFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &TraceLogFilterConfigFactory{}, nil
}
