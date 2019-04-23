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

package codec

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
)

// compare binary put and get
func TestBinary(t *testing.T) {
	var timeout int = -1
	// encode
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(timeout))
	// decode
	data := binary.BigEndian.Uint32(b)
	// signed to unsigned, overflow
	const maxOverFlow int = 4294967295
	if int(data) != maxOverFlow {
		t.Errorf("expected overflow")
	}
	// new decode, keeps the signed
	//var newData int32
	//buf := bytes.NewReader(b)
	//binary.Read(buf, binary.BigEndian, &newData)
	newData := int32(binary.BigEndian.Uint32(b))
	if int(newData) != timeout {
		t.Errorf("expected keep the signed int")
	}

}

func TestDecodeAndEncode_BoltV1(t *testing.T) {
	req := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    1,
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
	ctx := context.Background()
	buffer, err := BoltCodec.Encode(ctx, req)
	if err != nil {
		t.Fatal("Encode bolt v1 request failed", err)
	}
	ctx1 := context.Background()
	v, err := BoltCodec.Decode(ctx1, buffer)
	if err != nil {
		t.Fatal("Decode bolt v1 data failed", err)
	}
	req1, ok := v.(*sofarpc.BoltRequest)
	if !ok {
		t.Fatal("Decode bolt v1 request failed")
	}
	// verify
	// just verify timeout now
	if req.Timeout != req1.Timeout {
		t.Errorf("decode request is not equal origin request, origin: %d, got: %d", req.Timeout, req1.Timeout)
	}
}

func TestDecodeAndEncode_BoltV2(t *testing.T) {
	// just a request data for unit test
	// may be it is a invalid boltv2 request
	req := &sofarpc.BoltRequestV2{
		BoltRequest: sofarpc.BoltRequest{
			Protocol: sofarpc.PROTOCOL_CODE_V2,
			CmdType:  sofarpc.REQUEST,
			CmdCode:  sofarpc.RPC_REQUEST,
			Version:  0x02,
			ReqID:    1,
			Codec:    sofarpc.HESSIAN2_SERIALIZE,
			Timeout:  -1,
		},
		Version1:   0x01,
		SwitchCode: 0x00,
	}
	ctx := context.Background()
	buffer, err := BoltCodecV2.Encode(ctx, req)
	if err != nil {
		t.Fatal("Encode bolt v2 request failed", err)
	}
	ctx1 := context.Background()
	v, err := BoltCodecV2.Decode(ctx1, buffer)
	if err != nil {
		t.Fatal("Decode bolt v2 data failed", err)
	}
	req1, ok := v.(*sofarpc.BoltRequestV2)
	if !ok {
		t.Fatal("Decode bolt v2 request failed")
	}
	// verify
	// just verify timeout now
	if req.Timeout != req1.Timeout {
		t.Errorf("decode request is not equal origin request, origin: %d, got: %d", req.Timeout, req1.Timeout)
	}
}

func BenchmarkBoltCodec_Encode(b *testing.B) {
	request := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    256,
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	request.RequestHeader = map[string]string{"service": "com.alipay.test.sample.facade"} // used for sofa routing
	headerBuf := buffer.GetIoBuffer(128)

	if err := serialize.Instance.SerializeMap(request.RequestHeader, headerBuf); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBuf.Bytes()
		request.HeaderLen = int16(len(request.HeaderMap))
	}

	request.ClassName = []byte("com.alipay.sofa.hsf.SofaRequest")
	request.ClassLen = int16(len(request.ClassName))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BoltCodec.Encode(context.Background(), request)
	}
}

func BenchmarkBoltCodec_Decode(b *testing.B) {
	request := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    256,
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	request.RequestHeader = map[string]string{"service": "com.alipay.test.sample.facade"} // used for sofa routing
	headerBuf := buffer.GetIoBuffer(128)

	if err := serialize.Instance.SerializeMap(request.RequestHeader, headerBuf); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBuf.Bytes()
		request.HeaderLen = int16(len(request.HeaderMap))
	}

	request.ClassName = []byte("com.alipay.sofa.hsf.SofaRequest")
	request.ClassLen = int16(len(request.ClassName))
	buf, _ := BoltCodec.Encode(context.Background(), request)
	requestBytes := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BoltCodec.Decode(context.Background(), buffer.NewIoBufferBytes(requestBytes))
	}
}

func BenchmarkBoltCodec_All(b *testing.B) {
	request := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    256,
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	request.RequestHeader = map[string]string{"service": "com.alipay.test.sample.facade"} // used for sofa routing
	headerBuf := buffer.GetIoBuffer(128)

	if err := serialize.Instance.SerializeMap(request.RequestHeader, headerBuf); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBuf.Bytes()
		request.HeaderLen = int16(len(request.HeaderMap))
	}

	request.ClassName = []byte("com.alipay.sofa.hsf.SofaRequest")
	request.ClassLen = int16(len(request.ClassName))
	buf, _ := BoltCodec.Encode(context.Background(), request)
	requestBytes := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		req, _ := BoltCodec.Decode(ctx, buffer.NewIoBufferBytes(requestBytes))
		BoltCodec.Encode(ctx, req)
	}
}

// compare binary put and get
func BenchmarkBinaryOld(b *testing.B) {
	var timeout int = -1
	// encode
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(timeout))

	for i := 0; i < b.N; i++ {
		// decode
		_ = int32(binary.BigEndian.Uint32(buf))
	}
}

// compare binary put and get
func BenchmarkBinaryNew(b *testing.B) {
	var timeout int = -1
	// encode
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(timeout))

	for i := 0; i < b.N; i++ {
		// decode
		var newData int32
		buf := bytes.NewReader(buf)
		binary.Read(buf, binary.BigEndian, &newData)
	}
}
