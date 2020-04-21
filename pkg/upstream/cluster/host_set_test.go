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

package cluster

import (
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

func newSimpleMockHost(addr string, metaValue string) *mockHost {
	return &mockHost{
		addr: addr,
		meta: api.Metadata{
			"key": metaValue,
		},
	}
}

type simpleMockHostConfig struct {
	addr      string
	metaValue string
}

func TestHostSetDistinct(t *testing.T) {
	hs := &hostSet{}
	ip := "127.0.0.1"
	var hosts []types.Host
	for i := 0; i < 5; i++ {
		host := &mockHost{
			addr: ip,
		}
		hosts = append(hosts, host)
	}
	hs.setFinalHost(hosts)
	if len(hs.Hosts()) != 1 {
		t.Fatal("hostset distinct failed")
	}
}
