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
	RegisterLBType(types.WeightedRoundRobin, newSmoothWeightedRRLoadBalancer)
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

/*
SW (smoothWeightedRRLoadBalancer) is a struct that contains weighted items and provides methods to select a weighted item.
It is used for the smooth weighted round-robin balancing algorithm. This algorithm is implemented in Nginx:
https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35.
Algorithm is as follows: on each peer selection we increase current_weight
of each eligible peer by its weight, select peer with greatest current_weight
and reduce its current_weight by total number of weight points distributed
among peers.
*/
type smoothWeightedRRLoadBalancer struct {
	hosts         types.HostSet
	hostsWeighted map[string]*hostSmoothWeighted
}

type hostSmoothWeighted struct {
	weight          int
	currentWeight   int
	effectiveWeight int
}

func newSmoothWeightedRRLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	smoothWRRLoadBalancer := &smoothWeightedRRLoadBalancer{
		hosts:         hosts,
		hostsWeighted: make(map[string]*hostSmoothWeighted),
	}
	// iterate over all hosts to init host with Weighted
	for _, host := range hosts.Hosts() {
		w := int(host.Weight())
		smoothWRRLoadBalancer.hostsWeighted[host.AddressString()] = &hostSmoothWeighted{
			effectiveWeight: w,
			weight:          w,
		}
	}
	return smoothWRRLoadBalancer
}

// smooth weighted round robin
// O(n), traverse over all hosts
func (lb *smoothWeightedRRLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	totalWeight := 0
	var selectedHostWeighted *hostSmoothWeighted
	var selectedHost types.Host
	// TODO: lock?
	for _, host := range lb.hosts.Hosts() {
		if !host.Health() {
			continue
		}
		addr := host.AddressString()
		hw, ok := lb.hostsWeighted[addr]
		if !ok {
			continue
		}
		hw.currentWeight += hw.effectiveWeight
		totalWeight += hw.effectiveWeight

		if hw.effectiveWeight < hw.weight {
			hw.effectiveWeight++
		}

		if selectedHostWeighted == nil || hw.currentWeight > selectedHostWeighted.currentWeight {
			selectedHostWeighted = hw
			selectedHost = host
		}
	}
	//
	if selectedHostWeighted != nil {
		selectedHostWeighted.currentWeight -= totalWeight
	}
	return selectedHost
}

func (lb *smoothWeightedRRLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *smoothWeightedRRLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}
