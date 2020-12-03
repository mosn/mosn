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
	"github.com/stretchr/testify/assert"
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

func TestCLone(t *testing.T) {
	var kv = map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	var h = &Header{}
	for k, v := range kv {
		h.Set(k, v)
	}

	var hClone = h.Clone()
	for k, v := range kv {
		value, ok := hClone.Get(k)
		assert.True(t, ok)
		assert.Equal(t, value, v)
	}
}

func TestHeaderRange(t *testing.T) {
	var kv = map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	var h = &Header{}
	for k, v := range kv {
		h.Set(k, v)
	}

	h.Range(func(key, value string) bool {
		v, ok := kv[key]
		assert.True(t, ok)
		assert.Equal(t, v, value)
		return true
	})
}

// we should get the value we set in header
func TestHeaderSetGetDel(t *testing.T) {
	var h = &Header{}
	h.Set("kkk", "vvv")
	assert.True(t, h.Changed)

	v, ok := h.Get("kkk")
	assert.True(t, ok)
	assert.Equal(t, v, "vvv")

	// override the original value
	h.Set("kkk", "vvvv")
	v, _ = h.Get("kkk")
	assert.Equal(t, v, "vvvv")

	buf := buffer.NewIoBuffer(100)
	assert.Equal(t, 0, buf.Len())

	EncodeHeader(buf, h)
	assert.Less(t, 0, buf.Len())

	assert.Equal(t, GetHeaderEncodeLength(h), buf.Len())

	h.Del("kkk")
	_, ok = h.Get("kkk")
	assert.False(t, ok)
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
