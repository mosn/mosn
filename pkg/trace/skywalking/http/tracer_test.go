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
	"testing"
	"time"

	"github.com/SkyAPM/go2sky/propagation"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace/skywalking"
)

const header string = "1-MTU1NTY0NDg4Mjk2Nzg2ODAwMC4wLjU5NDYzNzUyMDYzMzg3NDkwODc=" +
	"-NS4xNTU1NjQ0ODgyOTY3ODg5MDAwLjM3NzUyMjE1NzQ0Nzk0NjM3NTg=" +
	"-1-2-3-I2NvbS5oZWxsby5IZWxsb1dvcmxk-Iy9yZXN0L2Fh-Iy9nYXRld2F5L2Nj"

func TestHttpSkyTracerStartFinish(t *testing.T) {
	driver := skywalking.NewSkyDriverImpl()
	driver.Register(protocol.HTTP1, NewHttpSkyTracer)

	// use default config
	err := driver.Init(nil)
	if err != nil {
		t.Error(err.Error())
	}

	tracer := driver.Get(protocol.HTTP1)
	if tracer == nil {
		t.Error("get tracer from driver failed")
	}

	// case 1: normal
	request := http.RequestHeader{&fasthttp.RequestHeader{}, nil}
	request.SetRequestURI("/test")
	request.SetHost("127.0.0.1:80")
	request.SetMethod("GET")
	request.Add(propagation.Header, header)

	requestInfo := network.NewRequestInfo()
	requestInfo.SetResponseCode(500)
	requestInfo.SetUpstreamLocalAddress("127.0.0.1:81")

	ctx := context.Background()
	s := tracer.Start(ctx, request, time.Now())
	s.InjectContext(request, requestInfo)
	s.SetRequestInfo(requestInfo)
	s.FinishSpan()

	sw6, ok := request.Get(propagation.Header)
	if !ok {
		t.Error("the request header was not injected into the upstream service")
	}
	if sw6 == header {
		t.Error("the request header for the downstream service injection was not removed")
	}

	// case 2: create entry span error
	s = tracer.Start(nil, request, time.Now())
	if s != skywalking.NoopSpan {
		t.Error("parameter errors were not handled correctly")
	}

	// case 3: create exit span
	requestInfo.SetUpstreamLocalAddress("")
	s = tracer.Start(ctx, request, time.Now())
	s.InjectContext(request, requestInfo)
	s.SetRequestInfo(requestInfo)
	s.FinishSpan()

	// case 4: create span error
	s = tracer.Start(ctx, ctx, time.Now())
	if s != skywalking.NoopSpan {
		t.Error("parameter errors were not handled correctly")
	}

	time.Sleep(time.Second)
}
