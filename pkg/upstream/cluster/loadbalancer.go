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
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// NewLoadBalancer can be register self defined type
var lbFactories map[types.LoadBalancerType]func(types.HostSet) types.LoadBalancer

func RegisterLBType(lbType types.LoadBalancerType, f func(types.HostSet) types.LoadBalancer) {
	if lbFactories == nil {
		lbFactories = make(map[types.LoadBalancerType]func(types.HostSet) types.LoadBalancer)
	}
	lbFactories[lbType] = f
}

var rrFactory *roundRobinLoadBalancerFactory

func init() {
	rrFactory = &roundRobinLoadBalancerFactory{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	RegisterLBType(types.RoundRobin, rrFactory.newRoundRobinLoadBalancer)
	RegisterLBType(types.Random, newRandomLoadBalancer)
	RegisterLBType(types.LeastActiveRequest, newleastActiveRequestLoadBalancer)
}

func NewLoadBalancer(lbType types.LoadBalancerType, hosts types.HostSet) types.LoadBalancer {
	if f, ok := lbFactories[lbType]; ok {
		return f(hosts)
	}
	return rrFactory.newRoundRobinLoadBalancer(hosts)
}

// LoadBalancer Implementations

type randomLoadBalancer struct {
	mutex sync.Mutex
	rand  *rand.Rand
	hosts types.HostSet
}

func newRandomLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	return &randomLoadBalancer{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		hosts: hosts,
	}
}

func (lb *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.Hosts()
	total := len(targets)
	if total == 0 {
		return nil
	}
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	idx := lb.rand.Intn(total)
	for i := 0; i < total; i++ {
		host := targets[idx]
		if host.Health() {
			return host
		}
		idx = (idx + 1) % total
	}
	return nil
}

func (lb *randomLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *randomLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

type roundRobinLoadBalancer struct {
	hosts   types.HostSet
	rrIndex uint32
}

type roundRobinLoadBalancerFactory struct {
	mutex sync.Mutex
	rand  *rand.Rand
}

func (f *roundRobinLoadBalancerFactory) newRoundRobinLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	var idx uint32
	hostsList := hosts.Hosts()
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if len(hostsList) != 0 {
		idx = f.rand.Uint32() % uint32(len(hostsList))
	}
	return &roundRobinLoadBalancer{
		hosts:   hosts,
		rrIndex: idx,
	}
}

func (lb *roundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.Hosts()
	total := len(targets)
	if total == 0 {
		return nil
	}
	for i := 0; i < total; i++ {
		index := atomic.AddUint32(&lb.rrIndex, 1) % uint32(total)
		host := targets[index]
		if host.Health() {
			return host
		}
	}
	return nil
}

func (lb *roundRobinLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *roundRobinLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

// leastActiveRequestLoadBalancer choose the host with the least active request
type leastActiveRequestLoadBalancer struct {
	hosts types.HostSet
	rand  *rand.Rand
}

func newleastActiveRequestLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	return &leastActiveRequestLoadBalancer{
		hosts: hosts,
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (lb *leastActiveRequestLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	healthHosts := lb.hosts.HealthyHosts()
	if len(healthHosts) == 0 {
		return nil
	}
	// exactly one healthy host, return this host directly
	if len(healthHosts) == 1 {
		return healthHosts[0]
	}
	// The list of hosts having the same least active request value
	candicate := make([]types.Host, 0, len(healthHosts))
	// The least active request value of all hosts
	leastActive := int64(math.MaxInt64)
	for _, host := range healthHosts {
		active := host.HostStats().UpstreamRequestActive.Count()
		// less than the current least active
		if active < leastActive {
			leastActive = active
			candicate = candicate[:0]
			candicate = append(candicate, host)
		} else if active == leastActive {
			candicate = append(candicate, host)
		}
	}
	//  exactly one host, return this host directly
	if len(candicate) == 1 {
		return candicate[0]
	}
	// choose one candicate based on the random
	return candicate[lb.rand.Intn(len(candicate))]
}

func (lb *leastActiveRequestLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *leastActiveRequestLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

// TODO:
// WRR
