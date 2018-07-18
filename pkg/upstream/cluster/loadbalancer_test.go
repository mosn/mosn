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

	"github.com/alipay/sofa-mosn/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func Test_roundRobinLoadBalancer_ChooseHost(t *testing.T) {

	host1 := NewHost(v2.Host{Address: "127.0.0.1", Hostname: "test", Weight: 0}, nil)
	host2 := NewHost(v2.Host{Address: "127.0.0.2", Hostname: "test2", Weight: 0}, nil)
	host3 := NewHost(v2.Host{Address: "127.0.0.3", Hostname: "test", Weight: 0}, nil)
	host4 := NewHost(v2.Host{Address: "127.0.0.4", Hostname: "test2", Weight: 0}, nil)
	host5 := NewHost(v2.Host{Address: "127.0.0.5", Hostname: "test2", Weight: 0}, nil)

	hosts1 := []types.Host{host1, host2}
	hosts2 := []types.Host{host3, host4}
	hosts3 := []types.Host{host5}

	hs1 := hostSet{
		hosts:        hosts1,
		healthyHosts: hosts1,
	}

	hs2 := hostSet{
		hosts:        hosts2,
		healthyHosts: hosts2,
	}

	hs3 := hostSet{
		hosts:        hosts3,
		healthyHosts: hosts3,
	}

	hostset := []types.HostSet{&hs1, &hs2, &hs3}

	prioritySet := prioritySet{
		hostSets: hostset,
	}

	loadbalaner := loadbalaner{
		prioritySet: &prioritySet,
	}

	l := &roundRobinLoadBalancer{
		loadbalaner: loadbalaner,
	}

	want := []types.Host{host1, host2, host3, host4, host5}

	for i := 0; i < len(want); i++ {
		got := l.ChooseHost(nil)
		if got != want[i] {
			t.Errorf("Test Error in case %d , got %+v, but want %+v,", i, got, want[i])
		}
	}
}
