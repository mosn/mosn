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

package http

import (
	"context"
	"strconv"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/SkyAPM/go2sky/reporter/grpc/common"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/trace/skywalking"
	"mosn.io/mosn/pkg/types"
)

var (
	// MIME header key s. The canonicalization converts the first letter and any letter following a hyphen to upper case;
	sw6Header = [2]string{propagation.Header, "Sw6"}
)

func init() {
	trace.RegisterTracerBuilder(skywalking.SkyDriverName, protocol.HTTP1, NewHttpSkyTracer)
}

type httpSkyTracer struct {
	*go2sky.Tracer
}

func NewHttpSkyTracer(_ map[string]interface{}) (types.Tracer, error) {
	return &httpSkyTracer{}, nil
}

func (tracer *httpSkyTracer) SetGO2SkyTracer(t *go2sky.Tracer) {
	tracer.Tracer = t
}

func (tracer *httpSkyTracer) Start(ctx context.Context, request interface{}, _ time.Time) types.Span {
	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Debugf("[SkyWalking] [tracer] [http1] unable to get request header, downstream trace ignored")
		return skywalking.NoopSpan
	}

	// create entry span (downstream)
	requestURI := string(header.RequestURI())
	entry, nCtx, err := tracer.CreateEntrySpan(ctx, requestURI, func() (sw6 string, err error) {
		for _, h := range sw6Header {
			sw6, ok = header.Get(h)
			if ok {
				// delete the sw6 header, otherwise the upstream service will receive two sw6 header
				header.Del(h)
				return sw6, err
			}
		}
		return
	})
	if err != nil {
		log.DefaultLogger.Errorf("[SkyWalking] [tracer] [http1] create entry span error, err: %v", err)
		return skywalking.NoopSpan
	}
	entry.Tag(go2sky.TagHTTPMethod, string(header.Method()))
	entry.Tag(go2sky.TagURL, string(header.Header())+requestURI)
	entry.SetComponent(skywalking.MOSNComponentID)
	entry.SetSpanLayer(common.SpanLayer_Http)

	return httpSkySpan{
		tracer: tracer,
		ctx:    nCtx,
		carrier: &spanCarrier{
			entrySpan: entry,
		},
	}
}

type spanCarrier struct {
	entrySpan go2sky.Span
	exitSpan  go2sky.Span
}

type httpSkySpan struct {
	skywalking.SkySpan
	tracer  *httpSkyTracer
	ctx     context.Context
	carrier *spanCarrier
}

func (h httpSkySpan) TraceId() string {
	return go2sky.TraceID(h.ctx)
}

func (h httpSkySpan) InjectContext(requestHeaders types.HeaderMap, requestInfo api.RequestInfo) {
	header, ok := requestHeaders.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Debugf("[SkyWalking] [tracer] [http1] unable to get request header, upstream trace ignored")
		return
	}
	requestURI := string(header.RequestURI())
	upstreamLocalAddress := requestInfo.UpstreamLocalAddress()

	// create exit span (upstream)
	exit, err := h.tracer.CreateExitSpan(h.ctx, requestURI, upstreamLocalAddress, func(header string) error {
		requestHeaders.Add(propagation.Header, header)
		return nil
	})
	if err != nil {
		log.DefaultLogger.Errorf("[SkyWalking] [tracer] [http1] create exit span error, err: %v", err)
		return
	}

	exit.SetComponent(skywalking.MOSNComponentID)
	exit.SetSpanLayer(common.SpanLayer_Http)
	h.carrier.exitSpan = exit
}

func (h httpSkySpan) SetRequestInfo(requestInfo api.RequestInfo) {
	responseCode := strconv.Itoa(requestInfo.ResponseCode())

	// end exit span (upstream)
	if h.carrier.exitSpan != nil {
		exit := h.carrier.exitSpan
		if requestInfo.ResponseCode() >= http.BadRequest {
			exit.Error(time.Now(), skywalking.ErrorLog)
		}
		exit.Tag(go2sky.TagStatusCode, responseCode)
		exit.End()
	}

	// entry span (downstream)
	entry := h.carrier.entrySpan
	if requestInfo.ResponseCode() >= http.BadRequest {
		entry.Error(time.Now(), skywalking.ErrorLog)
	}
	entry.Tag(go2sky.TagStatusCode, responseCode)
	// TODO More mosn information
}

func (h httpSkySpan) FinishSpan() {
	h.carrier.entrySpan.End()
}
