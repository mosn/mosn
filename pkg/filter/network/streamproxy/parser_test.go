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

import "testing"

func TestParseTCPProxy(t *testing.T) {
	m := map[string]interface{}{
		"stat_prefix":          "tcp_proxy",
		"cluster":              "cluster",
		"max_connect_attempts": 1000,
		"routes": []interface{}{
			map[string]interface{}{
				"cluster": "test",
				"SourceAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"DestinationAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"SourcePort":      "8080",
				"DestinationPort": "8080",
			},
		},
	}
	tcpproxy, err := ParseStreamProxy(m)
	if err != nil {
		t.Error(err)
		return
	}
	if len(tcpproxy.Routes) != 1 {
		t.Error("parse tcpproxy failed")
	} else {
		r := tcpproxy.Routes[0]
		if !(r.Cluster == "test" &&
			len(r.SourceAddrs) == 1 &&
			r.SourceAddrs[0].Address == "127.0.0.1" &&
			r.SourceAddrs[0].Length == 32 &&
			len(r.DestinationAddrs) == 1 &&
			r.DestinationAddrs[0].Address == "127.0.0.1" &&
			r.DestinationAddrs[0].Length == 32 &&
			r.SourcePort == "8080" &&
			r.DestinationPort == "8080") {
			t.Error("route failed")
		}
	}
}

func TestParseUDPProxy(t *testing.T) {
	m := map[string]interface{}{
		"stat_prefix":          "udp_proxy",
		"cluster":              "cluster",
		"max_connect_attempts": 1000,
		"routes": []interface{}{
			map[string]interface{}{
				"cluster": "test",
				"SourceAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"DestinationAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"SourcePort":      "8080",
				"DestinationPort": "8080",
			},
		},
	}
	udpproxy, err := ParseStreamProxy(m)
	if err != nil {
		t.Error(err)
		return
	}
	if len(udpproxy.Routes) != 1 {
		t.Error("parse udpproxy failed")
	} else {
		r := udpproxy.Routes[0]
		if !(r.Cluster == "test" &&
			len(r.SourceAddrs) == 1 &&
			r.SourceAddrs[0].Address == "127.0.0.1" &&
			r.SourceAddrs[0].Length == 32 &&
			len(r.DestinationAddrs) == 1 &&
			r.DestinationAddrs[0].Address == "127.0.0.1" &&
			r.DestinationAddrs[0].Length == 32 &&
			r.SourcePort == "8080" &&
			r.DestinationPort == "8080") {
			t.Error("route failed")
		}
	}
}
