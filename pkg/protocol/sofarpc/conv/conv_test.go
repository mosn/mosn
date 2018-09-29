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

package conv

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
)

var (
	bolt = new(sofarpc.BoltRequestCommand)

	http = make(map[string]string)

	className = "com.alipay.sofa.rpc.SofaRequset"
	headers   = map[string]string{
		"service": "com.alipay.test.service:1.0",
	}
)

func init() {
	bolt.Protocol = sofarpc.PROTOCOL_CODE_V1
	bolt.CmdType = sofarpc.REQUEST
	bolt.CmdCode = sofarpc.RPC_REQUEST
	bolt.Version = 0
	bolt.ReqID = 1
	bolt.CodecPro = sofarpc.HESSIAN2_SERIALIZE
	bolt.Timeout = time.Now().Nanosecond()

	bolt.RequestClass = className
	bolt.RequestHeader = headers

	bolt.ClassName, _ = serialize.Instance.Serialize(bolt.RequestClass)
	bolt.ClassLen = int16(len(bolt.ClassName))
	bolt.HeaderMap, _ = serialize.Instance.Serialize(bolt.RequestHeader)
	bolt.HeaderLen = int16(len(bolt.HeaderMap))

	http["service"] = "com.alipay.test.service:1.0"
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(bolt.Protocol), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(bolt.CmdType), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(bolt.CmdCode), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(bolt.Version), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(bolt.ReqID), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(bolt.CodecPro), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)] = strconv.FormatUint(uint64(bolt.Timeout), 10)
	//http[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(bolt.ClassLen), 10)
	//http[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(bolt.HeaderLen), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(bolt.ContentLen), 10)
	http[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = bolt.RequestClass
}

// http/x -> sofa
func TestH2SConvertHeader(t *testing.T) {

	if sofa, err := protocol.ConvertHeader(context.Background(), protocol.HTTP1, sofarpc.SofaRPC, protocol.CommonHeader(http)); err == nil {
		sofacmd := sofa.(*sofarpc.BoltRequestCommand)
		// serialize classname and header
		if sofacmd.RequestClass != "" {
			sofacmd.ClassName, _ = serialize.Instance.Serialize(sofacmd.RequestClass)
			sofacmd.ClassLen = int16(len(sofacmd.ClassName))
		}

		if sofacmd.RequestHeader != nil {
			sofacmd.HeaderMap, _ = serialize.Instance.Serialize(sofacmd.RequestHeader)
			sofacmd.HeaderLen = int16(len(sofacmd.HeaderMap))
		}
		if !reflect.DeepEqual(sofa, bolt) {
			t.Errorf("convert http/x -> sofa failed, result not equal")
		}

	} else {
		t.Errorf("convert http/x -> sofa failed, %s", err.Error())
	}
}

// sofa -> http/x
func TestS2HConvertHeader(t *testing.T) {

	if http, err := protocol.ConvertHeader(context.Background(), sofarpc.SofaRPC, protocol.HTTP1, bolt); err == nil {
		if !reflect.DeepEqual(http, http) {
			t.Errorf("convert sofa -> http/x failed, result not equal")
		}
	} else {
		t.Errorf("convert sofa -> http/x failed, %s", err.Error())
	}

}
