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

package dubbo

import (
	"bufio"
	"context"
	"fmt"
	"testing"

	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/api"
)

func Test_dubboProtocol_Hijack(t *testing.T) {
	type args struct {
		request    api.XFrame
		statusCode uint32
	}
	tests := []struct {
		name  string
		proto *dubboProtocol
		args  args
	}{
		{
			name:  "normal",
			proto: &dubboProtocol{},
			args: struct {
				request    api.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id: 1,
					},
				},
				statusCode: uint32(api.NoHealthUpstreamCode),
			},
		},
		{
			name:  "status not registry",
			proto: &dubboProtocol{},
			args: struct {
				request    api.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id: 1,
					},
				},
				statusCode: uint32(99),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := &dubboProtocol{}
			respFrame := proto.Hijack(context.TODO(), tt.args.request, tt.args.statusCode)

			buffer, err := encodeFrame(context.TODO(), respFrame.(*Frame))
			if err != nil {
				t.Errorf("%s-> test hijack encode fail:%s", tt.name, err)
				return
			}

			codecR := hessian.NewHessianCodec(bufio.NewReader(buffer))

			h := &hessian.DubboHeader{}
			if err = codecR.ReadHeader(h); err != nil {
				t.Errorf("%s-> hessian encode header fail:%s", tt.name, err)
			}
			if uint64(h.ID) != tt.args.request.GetRequestId() {
				t.Errorf("%s-> dubbo hijack response requestID not equal, want:%d, get:%d", tt.name, tt.args.request.GetRequestId(), h.ID)
				return
			}

			resp := &hessian.Response{}
			if err = codecR.ReadBody(resp); err != nil {
				t.Errorf("%s-> hessian encode body fail:%s", tt.name, err)
				return
			}

			expect, ok := dubboMosnStatusMap[int(tt.args.statusCode)]
			if !ok {
				expect = dubboStatusInfo{
					Status: hessian.Response_SERVICE_ERROR,
					Msg:    fmt.Sprintf("%d status not define", tt.args.statusCode),
				}
			}
			expect.Msg = fmt.Sprintf("java exception:%s", expect.Msg)

			if h.ResponseStatus != expect.Status || resp.Exception.Error() != expect.Msg {
				t.Errorf("%s-> dubbo hijack response or exception fail, input{responseStatus:%d, msg:%s}, output:{responseStatus:%d, msg:%s}", tt.name, tt.args.statusCode, expect.Msg, h.ResponseStatus, resp.Exception.Error())
			}
		})
	}
}
