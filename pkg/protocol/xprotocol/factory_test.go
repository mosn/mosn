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
	"testing"
)

func Test_Factory_Protocol(t *testing.T) {
	mp := &mockProtocol{}

	// 1. get nil
	proto := GetProtocol(mockProtocolName)
	if proto != nil {
		t.Error("got unexpected registered protocol:", mockProtocolName)
	}

	// 2. register
	err := RegisterProtocol(mockProtocolName, mp)
	if err != nil {
		t.Error("register protocol failed:", err)
	}

	// 3. get
	proto = GetProtocol(mockProtocolName)
	if proto == nil {
		t.Error("get protocol failed")
	}
}

func Test_Factory_Matcher(t *testing.T) {
	// 1. get nil
	matcher := GetMatcher(mockProtocolName)
	if matcher != nil {
		t.Error("got unexpected registered matcher:", mockProtocolName)
	}

	// 2. register
	err := RegisterMatcher(mockProtocolName, mockMatcher)
	if err != nil {
		t.Error("register matcher failed:", err)
	}

	// 3. get
	matcher = GetMatcher(mockProtocolName)
	if matcher == nil {
		t.Error("get matcher failed")
	}
}

func Test_Factory_Mapping(t *testing.T) {
	mm := &mockMapping{}

	// 1. get nil
	mapping := GetMapping(mockProtocolName)
	if mapping != nil {
		t.Error("got unexpected registered mapping:", mockProtocolName)
	}

	// 2. register
	err := RegisterMapping(mockProtocolName, mm)
	if err != nil {
		t.Error("register mapping failed:", err)
	}

	// 3. get
	mapping = GetMapping(mockProtocolName)
	if mapping == nil {
		t.Error("get mapping failed")
	}
}
