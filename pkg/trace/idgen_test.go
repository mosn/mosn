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

package trace

import "testing"

func TestIdGenerator_GenerateTraceId(t *testing.T) {
	traceId := IdGen().GenerateTraceId()
	if traceId == "" {
		t.Error("Generate traceId is empty")
	} else {
		t.Log(traceId)
	}
}

func TestIdGenerator_ipToHexString(t *testing.T) {
	ipHex := ipToHexString(GetIp())

	if ipHex == "" {
		t.Error("IP to Hex is empty.")
	}

	if len(ipHex) != 8 {
		t.Error("Wrong format of IP to Hex, the length should be 8")
	}
}
