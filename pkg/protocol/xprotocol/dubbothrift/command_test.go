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
	"reflect"
	"strconv"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/api"

	"mosn.io/mosn/pkg/protocol"

	"mosn.io/pkg/buffer"
)

func TestFrame(t *testing.T) {
	baseContent := baseData()
	frame := &Frame{
		Header: Header{
			Magic:        MagicTag,
			Id:           0,
			CommonHeader: protocol.CommonHeader{},
		},
		payload: []byte{0x4e, 0x4e},
	}
	frame.SetRequestId(1)
	frame.Magic = MagicTag
	//request
	frame.Direction = 1
	//set data
	frame.SetData(buffer.NewIoBufferBytes(baseContent))

	frame.Set(ServiceNameHeader, "com.pkg.test.TestService")
	frame.Set(MessageTypeNameHeader, strconv.Itoa(int(thrift.CALL)))

	if frame.GetRequestId() != 1 {
		t.Errorf("thrift method GetRequestId error")
	}
	if frame.IsHeartbeatFrame() != false {
		t.Errorf("thrift method IsHeartbeatFrame error")
	}
	if frame.GetStreamType() != api.Request {
		t.Errorf("thrift method GetStreamType error")
	}
	if frame.GetStatusCode() != 1 {
		t.Errorf("thrift method GetStatusCode error")
	}
	content := frame.GetData().Bytes()
	if !reflect.DeepEqual(content, baseData()) {
		t.Errorf("dubbo method GetData error")
	}
	byteFrame, error := encodeFrame(nil, frame)
	t.Log(byteFrame.Bytes())
	if error != nil || !reflect.DeepEqual(byteFrame.Bytes(), encodeData) {
		t.Errorf("dubbo encode freame panic:%s", error)
	}
	//decode the byte data
	want, err := decodeFrame(context.TODO(), buffer.NewIoBufferBytes(encodeData))
	if err != nil {
		t.Errorf("dubbo decode freame panic:%s", err)
		return
	}
	wantFrame, ok := want.(*Frame)
	if !ok {
		t.Errorf("dubbo decode freame error")
		return
	}
	//compare the content
	if !reflect.DeepEqual(wantFrame.GetData().Bytes(), baseData()) {
		t.Errorf("dubbo decode freame error, data not equal")
	}
}

func baseData() []byte {
	bufferBytes := buffer.NewIoBuffer(1024)
	transport := thrift.NewStreamTransportW(bufferBytes)
	defer transport.Close()
	protocol := thrift.NewTBinaryProtocolTransport(transport)
	protocol.WriteMessageBegin("testMethod", thrift.CALL, 1)
	protocol.WriteMessageEnd()
	protocol.Flush(nil)
	return bufferBytes.Bytes()
}

var encodeData = []uint8{
	0, 0, 0, 67, 218, 188, 0, 0, 0, 67, 0, 45, 1, 0, 0, 0, 24, 99, 111, 109, 46, 112, 107, 103, 46, 116, 101, 115, 116, 46, 84, 101, 115, 116, 83, 101, 114, 118, 105, 99, 101, 0, 0, 0, 0, 0, 0, 0, 1, 128, 1, 0, 1, 0, 0, 0, 10, 116, 101, 115, 116, 77, 101, 116, 104, 111, 100, 0, 0, 0, 1,
}
