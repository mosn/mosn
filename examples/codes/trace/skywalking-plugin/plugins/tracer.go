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

package main

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	language_agent "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"mosn.io/api"
	"mosn.io/pkg/log"
	"mosn.io/pkg/protocol/http"
)

var (
	// MIME header key s. The canonicalization converts the first letter and any letter following a hyphen to upper case;
	sw8Header = [2]string{propagation.Header, "Sw8"}

	// tracer init once
	lock            sync.RWMutex
	tracer          *go2sky.Tracer
	MOSNComponentID int32 = 5003
	ErrorLog              = "Errors on upstream service"
)

func TracerBuilder(conf map[string]interface{}) (api.Tracer, error) {
	lock.Lock()
	defer lock.Unlock()
	if tracer != nil {
		return &httpSkyTracer{
			Tracer: tracer,
		}, nil
	}

	var err error
	tracer, err := newGO2SkyTracer(conf)
	if err != nil {
		return nil, err
	}
	return &httpSkyTracer{
		Tracer: tracer,
	}, nil
}

type httpSkyTracer struct {
	*go2sky.Tracer
}

func (tracer *httpSkyTracer) Start(ctx context.Context, request interface{}, _ time.Time) api.Span {
	header, ok := request.(http.RequestHeader)
	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Debugf("[SkyWalking] [tracer] [http1] unable to get request header, downstream trace ignored")
		return nil
	}

	// create entry span (downstream)
	requestURI := string(header.RequestURI())
	entry, nCtx, err := tracer.CreateEntrySpan(ctx, requestURI, func() (sw8 string, err error) {
		for _, h := range sw8Header {
			sw8, ok = header.Get(h)
			if ok {
				// delete the sw8 header, otherwise the upstream service will receive two sw8 header
				header.Del(h)
				return sw8, err
			}
		}
		return
	})
	if err != nil {
		log.DefaultLogger.Errorf("[SkyWalking] [tracer] [http1] create entry span error, err: %v", err)
		return nil
	}
	entry.Tag(go2sky.TagHTTPMethod, string(header.Method()))
	entry.Tag(go2sky.TagURL, string(header.Host())+requestURI)
	entry.SetComponent(MOSNComponentID)
	entry.SetSpanLayer(language_agent.SpanLayer_Http)

	return httpSkySpan{
		tracer: tracer,
		ctx:    nCtx,
		carrier: &SpanCarrier{
			EntrySpan: entry,
		},
	}
}

type httpSkySpan struct {
	SkySpan
	tracer  *httpSkyTracer
	ctx     context.Context
	carrier *SpanCarrier
}

func (h httpSkySpan) TraceId() string {
	return go2sky.TraceID(h.ctx)
}

func (h httpSkySpan) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
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

	exit.SetComponent(MOSNComponentID)
	exit.SetSpanLayer(language_agent.SpanLayer_Http)
	h.carrier.ExitSpan = exit
}

func (h httpSkySpan) SetRequestInfo(requestInfo api.RequestInfo) {
	responseCode := strconv.Itoa(requestInfo.ResponseCode())

	// end exit span (upstream)
	if h.carrier.ExitSpan != nil {
		exit := h.carrier.ExitSpan
		if requestInfo.ResponseCode() >= http.BadRequest {
			exit.Error(time.Now(), ErrorLog)
		}
		exit.Tag(go2sky.TagStatusCode, responseCode)
		exit.End()
	}

	// entry span (downstream)
	entry := h.carrier.EntrySpan
	if requestInfo.ResponseCode() >= http.BadRequest {
		entry.Error(time.Now(), ErrorLog)
	}
	entry.Tag(go2sky.TagStatusCode, responseCode)
	// TODO More mosn information
}

func (h httpSkySpan) FinishSpan() {
	h.carrier.EntrySpan.End()
}
