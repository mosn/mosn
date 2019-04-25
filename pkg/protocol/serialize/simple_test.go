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
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"testing"
)

func BenchmarkSerializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf := buffer.GetIoBuffer(128)
		Instance.SerializeMap(headers, buf)
	}
}

func BenchmarkDeserializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	buf := buffer.GetIoBuffer(128)
	Instance.SerializeMap(headers, buf)

	bytes := buf.Bytes()
	header := make(map[string]string)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Instance.DeserializeMap(bytes, header)
	}
}
