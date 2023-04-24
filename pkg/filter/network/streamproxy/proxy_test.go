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

package streamproxy

import (
	"net"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func Test_IpRangeList_Contains(t *testing.T) {
	ipRangeList := &IpRangeList{
		cidrRanges: []v2.CidrRange{
			*v2.Create("127.0.0.1", 24),
		},
	}
	httpAddr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !ipRangeList.Contains(httpAddr) {
		t.Errorf("test  ip range fail")
	}
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !ipRangeList.Contains(udpAddr) {
		t.Errorf("test  ip range fail")
	}
}

func Test_ParsePortRangeList(t *testing.T) {
	prList := ParsePortRangeList("80,443,8080-8089")
	httpPort := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !prList.Contains(httpPort) {
		t.Errorf("test http port fail")
	}
	udpPort := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !prList.Contains(udpPort) {
		t.Errorf("test http port fail")
	}
	randomPort := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8083,
	}
	if !prList.Contains(randomPort) {
		t.Errorf("test  port range fail")
	}
	randomUDPPort := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8083,
	}
	if !prList.Contains(randomUDPPort) {
		t.Errorf("test  port range fail")
	}
}
