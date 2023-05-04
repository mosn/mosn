package dubbothrift

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

import (
	"context"
	"testing"

	"mosn.io/pkg/buffer"
)

func TestDecodeFramePanic(t *testing.T) {
	data := buffer.NewIoBufferBytes(errData)

	_, err := decodeFrame(context.TODO(), data)
	if err != nil {
		t.Logf("recover thrift decode panic:%s", err)
		return
	}
}

func TestDecode(t *testing.T) {
	data := buffer.NewIoBufferBytes(data)
	// decode attachement

	_, err := decodeFrame(context.TODO(), data)
	if err != nil {
		t.Errorf("dubbo decode [cheap] panic:%s", err)
		return
	}
}

func BenchmarkDecode(t *testing.B) {
	ctx := context.TODO()
	for i := 0; i < t.N; i++ {
		data := buffer.NewIoBufferBytes(data)
		_, err := decodeFrame(ctx, data)
		if err != nil {
			t.Errorf("recover dubbo decode panic:%s", err)
		}
	}
}

var errData = []byte{
	0x99, 0xda, 0xbc, 0xc2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x04, 0x45,
	0x05, 0x32, 0x2e, 0x30, 0x2e, 0x32, 0x30, 0x2f, 0x63, 0x6f, 0x6d, 0x2e, 0x64, 0x6d, 0x61, 0x6c,
	0x6c, 0x2e, 0x64, 0x73, 0x66, 0x2e, 0x7a, 0x6f, 0x6e, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x78, 0x79,
	0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x78, 0x79, 0x49, 0x6e,
	0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x05, 0x30, 0x2e, 0x30, 0x2e, 0x30, 0x0e, 0x75, 0x6e,
	0x69, 0x74, 0x54, 0x65, 0x73, 0x74, 0x42, 0x79, 0x55, 0x6e, 0x69, 0x74, 0x30, 0x5f, 0x4c, 0x6a,
}

var data = []byte{
	0, 0, 0, 67, 218, 188, 0, 0, 0, 67, 0, 45, 1, 0, 0, 0, 24, 99, 111, 109, 46, 112, 107, 103, 46, 116, 101, 115, 116, 46, 84, 101, 115, 116, 83, 101, 114, 118, 105, 99, 101, 0, 0, 0, 0, 0, 0, 0, 1, 128, 1, 0, 1, 0, 0, 0, 10, 116, 101, 115, 116, 77, 101, 116, 104, 111, 100, 0, 0, 0, 1,
}
