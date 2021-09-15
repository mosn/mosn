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

package v2

import (
	"net"
	"testing"
)

func Test_Create(t *testing.T) {
	address := "127.0.0.1"
	length := uint32(32)
	ipRange := Create(address, length)
	if !(ipRange.Length == length && ipRange.Address == address) {
		t.Error("create ip 1 range not match")
	}
}

func Test_IsInRange(t *testing.T) {
	address := "192.168.0.1"
	length := uint32(24)
	ipRange := Create(address, length)
	if !(ipRange.Length == length && ipRange.Address == "192.168.0.1") {
		t.Error("create ip range not match")
	}
	if !ipRange.IsInRange(net.ParseIP("192.168.0.128")) {
		t.Error("test ip range fail")
	}
	if ipRange.IsInRange(net.ParseIP("192.168.1.128")) {
		t.Error("test ip range fail")
	}
}
