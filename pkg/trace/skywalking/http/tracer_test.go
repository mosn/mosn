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
	"mosn.io/api"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/trace/skywalking"
)

const header = "1-MWYyZDRiZjQ3YmY3MTFlYWI3OTRhY2RlNDgwMDExMjI=-MWU3YzIwNGE3YmY3MTFlYWI4NThhY2RlNDgwMDExMjI=" +
	"-0-c2VydmljZQ==-aW5zdGFuY2U=-cHJvcGFnYXRpb24=-cHJvcGFnYXRpb246NTU2Ng=="

func Test_httpSkyTraceStartAndFinish(t *testing.T) {
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

	type args struct {
		requestFunc     func() interface{}
		requestInfoFunc func() api.RequestInfo
		context         context.Context
	}
	tests := []struct {
		name         string
		args         args
		wantNoopSpan bool
		wantHeader   bool
	}{
		{
			name: "normal",
			args: args{
				requestFunc: func() interface{} {
					re := http.RequestHeader{
						&fasthttp.RequestHeader{},
					}
					re.SetRequestURI("/test")
					re.SetHost("127.0.0.1:80")
					re.SetMethod("GET")
					re.Add(propagation.Header, header)
					return re
				},
				requestInfoFunc: func() api.RequestInfo {
					reqInfo := network.NewRequestInfo()
					reqInfo.SetResponseCode(500)
					reqInfo.SetUpstreamLocalAddress("127.0.0.1:81")
					return reqInfo
				},
				context: context.Background(),
			},
			wantNoopSpan: false,
			wantHeader:   true,
		},
		{
			name: "create entry span error1",
			args: args{
				requestFunc: func() interface{} {
					re := http.RequestHeader{
						&fasthttp.RequestHeader{},
					}
					re.SetRequestURI("/test")
					re.SetHost("127.0.0.1:80")
					re.SetMethod("GET")
					re.Add(propagation.Header, header)
					return re
				},
				requestInfoFunc: func() api.RequestInfo {
					reqInfo := network.NewRequestInfo()
					reqInfo.SetResponseCode(500)
					reqInfo.SetUpstreamLocalAddress("127.0.0.1:81")
					return reqInfo
				},
			},
			wantNoopSpan: true,
		},
		{
			name: "create entry span error2",
			args: args{
				requestFunc: func() interface{} {
					return context.Background()
				},
				requestInfoFunc: func() api.RequestInfo {
					reqInfo := network.NewRequestInfo()
					reqInfo.SetResponseCode(500)
					reqInfo.SetUpstreamLocalAddress("127.0.0.1:81")
					return reqInfo
				},
				context: context.Background(),
			},
			wantNoopSpan: true,
		},
		{
			name: "create exit span error",
			args: args{
				requestFunc: func() interface{} {
					re := http.RequestHeader{
						&fasthttp.RequestHeader{},
					}
					re.SetRequestURI("/test")
					re.SetHost("127.0.0.1:80")
					re.SetMethod("GET")
					re.Add(propagation.Header, header)
					return re
				},
				requestInfoFunc: func() api.RequestInfo {
					reqInfo := network.NewRequestInfo()
					reqInfo.SetResponseCode(500)
					reqInfo.SetUpstreamLocalAddress("")
					return reqInfo
				},
				context: context.Background(),
			},
			wantNoopSpan: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downStreamRequest := tt.args.requestFunc()

			s := tracer.Start(tt.args.context, downStreamRequest, time.Now())

			upStreamRequest := http.RequestHeader{
				&fasthttp.RequestHeader{},
			}
			requestInfo := tt.args.requestInfoFunc()
			s.InjectContext(upStreamRequest, requestInfo)

			s.SetRequestInfo(requestInfo)
			s.FinishSpan()

			if tt.wantNoopSpan && s != skywalking.NoopSpan {
				t.Error("parameter errors were not handled correctly")
			}
			if !tt.wantNoopSpan && s == skywalking.NoopSpan {
				t.Errorf("not consistent with expected data")
			}

			_, ok := upStreamRequest.Get(propagation.Header)
			if tt.wantHeader && !ok {
				t.Error("the request header was not injected into the upstream service")
			}
			if !tt.wantHeader && ok {
				t.Errorf("not consistent with expected data")
			}
		})
	}
}
