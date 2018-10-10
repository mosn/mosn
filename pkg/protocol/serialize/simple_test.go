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

package serialize

import (
	"testing"
)

func TestNotEnoughBytes(t *testing.T) {
	b := make([]byte, 3)
	b[0] = 255
	b[1] = 255
	b[2] = 255
	_, err := readInt32(b)
	if err.Error() != "no enough bytes" {
		t.Error("Expect error")
	}
}

func TestReadIntMinusOne(t *testing.T) {
	b := make([]byte, 4)
	b[0] = 255
	b[1] = 255
	b[2] = 255
	b[3] = 255
	result, _ := readInt32(b)
	if result != -1 {
		t.Error("Decode error")
	}
}

func TestReadIntZero(t *testing.T) {
	b := make([]byte, 4)
	b[0] = 0
	b[1] = 0
	b[2] = 0
	b[3] = 0
	result, _ := readInt32(b)
	if result != 0 {
		t.Error("Decode error")
	}
}

func TestReadIntOne(t *testing.T) {
	b := make([]byte, 4)
	b[0] = 0
	b[1] = 0
	b[2] = 0
	b[3] = 1
	result, _ := readInt32(b)
	if result != 1 {
		t.Error("Decode error")
	}
}

func BenchmarkSerializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	for n := 0; n < b.N; n++ {
		Instance.Serialize(headers)
	}
}

func BenchmarkSerializeString(b *testing.B) {
	className := "com.alipay.sofa.rpc.core.request.SofaRequest"

	for n := 0; n < b.N; n++ {
		Instance.Serialize(className)
	}
}

func BenchmarkDeSerializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	buf, _ := Instance.Serialize(headers)
	header := make(map[string]string)

	for n := 0; n < b.N; n++ {
		Instance.DeSerialize(buf, &header)
	}
}

func BenchmarkDeSerializeString(b *testing.B) {
	className := "com.alipay.sofa.rpc.core.request.SofaRequest"
	buf, _ := Instance.Serialize(className)
	var class string

	for n := 0; n < b.N; n++ {
		Instance.DeSerialize(buf, &class)
	}
}
