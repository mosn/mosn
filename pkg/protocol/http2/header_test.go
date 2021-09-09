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

package http2

import (
	"testing"

	"mosn.io/mosn/pkg/protocol"
)

func TestHeader(t *testing.T) {
	header := protocol.CommonHeader(make(map[string]string))
	header.Set("a", "b")
	header.Set("c", "d")
	httpHeader := EncodeHeader(header)
	if httpHeader.Get("a") != "b" {
		t.Errorf("EncodeHeader error")
	}
	if httpHeader.Get("c") != "d" {
		t.Error("EncodeHeader error")
	}
}
