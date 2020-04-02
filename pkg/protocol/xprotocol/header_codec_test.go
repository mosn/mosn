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

package xprotocol

import (
	"encoding/hex"
	"testing"

	"mosn.io/pkg/buffer"
)

func BenchmarkEncodeHeader(b *testing.B) {
	h := &Header{}
	h.Set("service", "io.mosn.test.TestService:1.0")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		buf := buffer.GetIoBuffer(128)
		EncodeHeader(buf, h)
		buf.Reset()
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	h := &Header{}
	h.Set("service", "io.mosn.test.TestService:1.0")
	buf := buffer.GetIoBuffer(128)
	EncodeHeader(buf, h)

	decoded := &Header{}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		DecodeHeader(buf.Bytes(), decoded)
	}
}

func TestDecodeHeader(t *testing.T) {
	type args struct {
		b []byte
		m *Header
	}
	decodeString, _ := hex.DecodeString("0000000161FFFFFFFF00000001620000000163")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "empty", args: args{
			b: decodeString,
			m: &Header{},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DecodeHeader(tt.args.b, tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("DeserializeMap() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if val, ok := tt.args.m.Get("b"); !ok || val != "c" {
					t.Errorf("DeserializeMap() error = %v, wantErr %v", err, tt.wantErr)
				}
				if _, ok := tt.args.m.Get("a"); ok {
					t.Errorf("DeserializeMap() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
