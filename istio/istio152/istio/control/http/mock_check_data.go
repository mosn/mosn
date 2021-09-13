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

package http

import (
	"github.com/gogo/protobuf/jsonpb"
	"istio.io/api/mixer/v1"
)

type mockCheckData struct {
	istioAtttibutes        *v1.Attributes
	extractIstioAttributes int
	getSourceIPPort        int
}

func newMockCheckData() CheckData {
	return &mockCheckData{}
}

// ExtractIstioAttributes Find ex-istio-attributes" HTTP header.
// If found, base64 decode its value,  pass it out
func (c *mockCheckData) ExtractIstioAttributes() (data string, ret bool) {
	c.extractIstioAttributes++
	mar := jsonpb.Marshaler{}
	data, _ = mar.MarshalToString(c.istioAtttibutes)
	ret = true
	return
}

// GetSourceIPPort get downstream tcp connection ip and port.
func (c *mockCheckData) GetSourceIPPort() (ip string, port int32, ret bool) {
	c.getSourceIPPort++
	return
}
