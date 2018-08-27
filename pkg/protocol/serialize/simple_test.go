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
