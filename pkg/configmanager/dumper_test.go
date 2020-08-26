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

package configmanager

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/utils"
)

func TestDumpConfig(t *testing.T) {
	// prepare
	tmp := "/tmp/test_dump.json"
	err := utils.WriteFileSafety(tmp, []byte(`{}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_ = Load(tmp)
	// modify
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	config.Servers = []v2.ServerConfig{
		{
			ServerName: "test",
			GracefulTimeout: api.DurationConfig{
				Duration: time.Second,
			},
			Listeners: []v2.Listener{
				{
					ListenerConfig: v2.ListenerConfig{
						Network: "udp",
					},
					Addr: addr,
				},
			},
		},
	}
	// Test
	setDump()
	DumpConfig()
	// Verify
	b, err := ioutil.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	result := &v2.MOSNConfig{}
	if err := json.Unmarshal(b, result); err != nil {
		t.Fatal(err)
	}
	if !(result.Servers[0].Listeners[0].AddrConfig == "127.0.0.1:8080" &&
		result.Servers[0].Listeners[0].Network == "udp") {
		t.Fatalf("result is not expected : %+v", result)
	}
}
