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

package dubbothrift

import (
	"context"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
)

func Test_dubboProtocol_Hijack(t *testing.T) {
	type args struct {
		request    api.XFrame
		statusCode uint32
	}
	tests := []struct {
		name  string
		proto *thriftProtocol
		args  args
	}{
		{
			name:  "normal",
			proto: &thriftProtocol{},
			args: struct {
				request    api.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id:           1,
						CommonHeader: protocol.CommonHeader{ServiceNameHeader: "com.pkg.test.TestService", SeqIdNameHeader: "1", MethodNameHeader: "testMethod"},
					},
				},
				statusCode: uint32(api.NoHealthUpstreamCode),
			},
		},
		{
			name:  "status not registry",
			proto: &thriftProtocol{},
			args: struct {
				request    api.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id:           1,
						CommonHeader: protocol.CommonHeader{ServiceNameHeader: "com.pkg.test.TestService", SeqIdNameHeader: "1", MethodNameHeader: "testMethod"},
					},
				},
				statusCode: uint32(99),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := &thriftProtocol{}
			respFrame := proto.Hijack(context.TODO(), tt.args.request, tt.args.statusCode)

			enBuffer, err := encodeFrame(context.TODO(), respFrame.(*Frame))
			if err != nil {
				t.Errorf("%s-> test hijack encode fail:%s", tt.name, err)
				return
			}
			t.Log(enBuffer)

			transport := thrift.NewStreamTransportR(enBuffer)
			defer transport.Close()
			protocol := thrift.NewTBinaryProtocolTransport(transport)

			msgLen, err := protocol.ReadI32()

			if err != nil {
				t.Errorf("%s ->dubbothrift hijack response decode message length fail: %s", tt.name, err)
			}
			t.Logf("message length is : %d", msgLen)

			magic, err := protocol.ReadI16()

			if err != nil {
				t.Errorf("%s ->dubbothrift hijack response decode magic tag fail: %s", tt.name, err)
			}
			t.Logf("magic is : %v", magic)

			//message len
			protocol.ReadI32()
			//header len
			protocol.ReadI16()
			//version
			protocol.ReadByte()

			serviceName, err := protocol.ReadString()
			if err != nil {
				t.Errorf("%s ->dubbothrift hijack response decode serviceName fail: %s", tt.name, err)
			}
			wantServiceName, _ := tt.args.request.GetHeader().Get(ServiceNameHeader)
			if serviceName != wantServiceName {
				t.Errorf("%s->dubbothrift hijack response serviceName not equal, want:%s, get:%s", tt.name, wantServiceName, serviceName)
			}

			requestId, err := protocol.ReadI64()
			if err != nil {
				t.Errorf("%s ->dubbothrift hijack response decode requestId fail: %s", tt.name, err)
			}
			if uint64(requestId) != tt.args.request.GetRequestId() {
				t.Errorf("%s->dubbothrift hijack response requestID not equal, want:%d, get:%d", tt.name, tt.args.request.GetRequestId(), requestId)
			}

			name, typeId, seqId, err := protocol.ReadMessageBegin()

			t.Logf("name: %s, type: %v, seqId: %v", name, typeId, seqId)

			exception := thrift.NewTApplicationException(0, "")
			err = exception.Read(protocol)
			if err != nil {
				t.Errorf("%s->dubbothrift hijack response decode exception fail:%s", tt.name, err)
			}

			thriftErr, ok := dubboMosnStatusMap[int(tt.args.statusCode)]
			if !ok {
				thriftErr = dThriftStatusInfo{
					Status: thrift.UNKNOWN_APPLICATION_EXCEPTION,
					Msg:    "unknown application exception",
				}
			}

			errMsg := exception.Error()
			mType := exception.TypeId()

			if errMsg != thriftErr.Msg || mType != thriftErr.Status {
				t.Errorf("%s->dubbothrift hijack response exception status or msg error. expect: %v,%v, get: %v,%v", tt.name, thriftErr.Status, thriftErr.Msg, errMsg, mType)
			}

			t.Logf("terr: %v, err: %v, mType %v", thriftErr, errMsg, mType)
		})
	}
}
