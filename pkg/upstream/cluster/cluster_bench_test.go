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

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/types"
)

// Test Cluster Update Host
func BenchmarkClusterUpdateHost(b *testing.B) {
	cluster := _createTestCluster()
	// assume cluster have 1000 hosts
	pool := makePool(1010)
	hosts := make([]types.Host, 0, 1000)
	metas := []v2.Metadata{
		v2.Metadata{"version": "1", "zone": "a"},
		v2.Metadata{"version": "1", "zone": "b"},
		v2.Metadata{"version": "2", "zone": "a"},
	}
	for _, meta := range metas {
		hosts = append(hosts, pool.MakeHosts(300, meta)...)
	}
	hosts = append(hosts, pool.MakeHosts(100, nil)...)
	// hosts changes, some are removed, some are added
	var newHosts []types.Host
	for idx := range metas {
		newHosts = append(newHosts, hosts[idx:idx*300+5]...)
	}
	newHosts = append(newHosts, pool.MakeHosts(10, v2.Metadata{
		"version": "3",
		"zone":    "b",
	})...)
	b.Run("UpdateClusterHost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cluster.UpdateHosts(hosts)
			b.StartTimer()
			cluster.UpdateHosts(newHosts)
		}
	})
}
