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

package bolt2http

import (
	"context"
	"fmt"
	"github.com/basgys/goxml2json"
	"github.com/valyala/fasthttp"
	v2 "mosn.io/mosn/pkg/api/v2"
	"mosn.io/mosn/pkg/buffer"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"net/http"
	"strings"
)

// transcodeFilter is an implement of types.StreamReceiverFilter/types.StreamSendFilter
type transcodeFilter struct {
	ctx context.Context
	cfg *config

	needTranscode bool

	receiveHandler types.StreamReceiverFilterHandler
	sendHandler    types.StreamSenderFilterHandler
}

func newTranscodeFilter(ctx context.Context, cfg *config) *transcodeFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new payload limit filter")
	}
	return &transcodeFilter{
		ctx: ctx,
		cfg: cfg,
	}
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *transcodeFilter) readPerRouteConfig(ctx context.Context, cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	if payloadLimit, ok := cfg[v2.MosnBoltHttpTranscoder]; ok {
		if config, err := parseConfig(payloadLimit); err == nil {
			if log.Proxy.GetLogLevel() >= log.DEBUG {
				log.Proxy.Debugf(ctx, "use router config to replace stream filter config, config: %v", payloadLimit)
			}
			f.cfg = config
		}
	}
}

func (f *transcodeFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.receiveHandler = handler
}

func (f *transcodeFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	// check request
	if protocol := mosnctx.Get(ctx, types.ContextSubProtocol); protocol != string(bolt.ProtocolName) {
		return types.StreamFilterContinue
	}

	// for response check
	f.needTranscode = true

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "mosn.bolt2http stream filter receive request: %+v", headers)
	}
	if route := f.receiveHandler.Route(); route != nil {
		// TODO: makes ReadPerRouteConfig as the StreamReceiverFilter's function
		f.readPerRouteConfig(ctx, route.RouteRule().PerFilterConfig())
	}

	// do transcoding
	outHeaders, outBuf, outTrailers, err := f.transcodeRequest(headers, buf, trailers)
	if err != nil {
		log.Proxy.Errorf(ctx, "transcode failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(types.RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusBadRequest, headers)
		return types.StreamFilterStop
	}
	f.receiveHandler.SetRequestHeaders(outHeaders)
	f.receiveHandler.SetRequestData(outBuf)
	f.receiveHandler.SetRequestTrailers(outTrailers)
	return types.StreamFilterContinue
}

func (f *transcodeFilter) OnDestroy() {}

func (f *transcodeFilter) transcodeRequest(headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error) {
	sourceRequest := headers.(*bolt.Request)
	targetRequest := fasthttp.Request{}

	// header convert, use service as host and method as path(if exists)
	if service, ok := sourceRequest.Get("service"); ok {
		targetRequest.URI().SetHost(service)
	}

	if method, ok := sourceRequest.Get("method"); ok {
		targetRequest.URI().SetPath(method)
	}

	if sourceRequest.GetData() != nil && sourceRequest.GetData().Len() > 0 {
		targetRequest.Header.SetMethod("POST")
	} else {
		targetRequest.Header.SetMethod("GET")
	}

	sourceRequest.Range(func(Key, Value string) bool {
		targetRequest.Header.Set(Key, Value)
		return true
	})

	// payload convert
	if f.cfg.sourceEncoding == "xml" && f.cfg.targetEncoding == "json" {
		payloadReader := strings.NewReader(buf.String())
		json, err := xml2json.Convert(payloadReader)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("convert encoding from xml to json failed, payload: %+v", buf.String())
		}
		return mosnhttp.RequestHeader{RequestHeader: &targetRequest.Header}, buffer.NewIoBufferBytes(json.Bytes()), trailers, nil
	} else {
		return nil, nil, nil, fmt.Errorf("config not supported: %+v", f.cfg)
	}

	// return with no payload convert
	return mosnhttp.RequestHeader{RequestHeader: &targetRequest.Header}, buf, trailers, nil
}

// SetSenderFilterHandler sets the StreamSenderFilterHandler
func (f *transcodeFilter) SetSenderFilterHandler(handler types.StreamSenderFilterHandler) {
	f.sendHandler = handler
}

// Append encodes request/response
func (f *transcodeFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	if !f.needTranscode {
		return types.StreamFilterContinue
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "mosn.bolt2http stream filter receive response: %+v", headers)
	}

	// do transcoding
	outHeaders, outBuf, outTrailers, err := f.transcodeResponse(headers, buf, trailers)
	if err != nil {
		log.Proxy.Errorf(ctx, "transcode failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(types.RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusInternalServerError, headers)
		return types.StreamFilterStop
	}
	f.sendHandler.SetResponseHeaders(outHeaders)
	f.sendHandler.SetResponseData(outBuf)
	f.sendHandler.SetResponseTrailers(outTrailers)
	return types.StreamFilterContinue
}

func (f *transcodeFilter) transcodeResponse(headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error) {
	sourceResponse := headers.(mosnhttp.ResponseHeader)
	targetResponse := &bolt.Response{}

	boltProtocol := xprotocol.GetProtocol(bolt.ProtocolName)

	// header convert, status code
	sourceResponse.Range(func(Key, Value string) bool {
		targetResponse.Header.Set(Key, Value)
		return true
	})

	targetResponse.ResponseStatus = uint16(boltProtocol.Mapping(uint32(sourceResponse.StatusCode())))

	// payload convert
	if f.cfg.sourceEncoding == "xml" && f.cfg.targetEncoding == "json" {
		// TODO: convert response payload from json to xml
		targetResponse.Content = buf
	} else {
		return nil, nil, nil, fmt.Errorf("config not supported: %+v", f.cfg)
	}

	// return with no payload convert
	return targetResponse, buf, trailers, nil
}
