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
	"context"
	"reflect"
	"testing"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

func TestFrame(t *testing.T) {
	baseContent := []byte("test")
	frame := &Frame{
		Header: Header{
			Magic:   MagicTag,
			Flag:    0x22,
			Status:  0x00,
			Id:      0,
			DataLen: 0,
		},
		payload: []byte{0x4e, 0x4e},
	}
	frame.SetRequestId(1)
	frame.Magic = MagicTag
	//not event
	frame.IsEvent = false
	//request
	frame.Direction = 1
	//set data
	frame.SetData(buffer.NewIoBufferBytes(baseContent))
	frame.Status = 0x14

	if frame.GetRequestId() != 1 {
		t.Errorf("dubbo method GetRequestId error")
	}
	if frame.IsHeartbeatFrame() != false {
		t.Errorf("dubbo method IsHeartbeatFrame error")
	}
	if frame.GetStreamType() != api.Request {
		t.Errorf("dubbo method GetStreamType error")
	}
	if frame.GetStatusCode() != 0x14 {
		t.Errorf("dubbo method GetStatusCode error")
	}
	content := frame.GetData().Bytes()
	if !reflect.DeepEqual(content, baseContent) {
		t.Errorf("dubbo method GetData error")
	}
	byteFrame, error := encodeFrame(nil, frame)
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
	if string(wantFrame.GetData().Bytes()) != "test" {
		t.Errorf("dubbo decode freame error, data not equal")
	}
}

var encodeData = []uint8{
	218, 187, 34, 20, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 4, 116, 101, 115, 116,
}
