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

	"github.com/alipay/sofa-mosn/pkg/types"
)

// NewLoadBalancer
// Note: Random is the default lb
// Round Robin is realized as Weighted Round Robin
func NewLoadBalancer(lbType types.LoadBalancerType, prioritySet types.PrioritySet) types.LoadBalancer {
	switch lbType {
	case types.RoundRobin:
		return newSmoothWeightedRRLoadBalancer(prioritySet)
	default:
		return newRandomLoadbalancer(prioritySet)
	}
}

type loadbalancer struct {
	prioritySet types.PrioritySet
}

type randomLoadBalancer struct {
	loadbalancer
	randInstance *rand.Rand
	randMutex    sync.Mutex
}

func newRandomLoadbalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &randomLoadBalancer{
		loadbalancer: loadbalancer{
			prioritySet: prioritySet,
		},
		randInstance: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (l *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	hostSets := l.prioritySet.HostSetsByPriority()
	if len(hostSets) == 0 {
		return nil
	}

	l.randMutex.Lock()
	defer l.randMutex.Unlock()
	idx := l.randInstance.Intn(len(hostSets))
	hostset := hostSets[idx]
	hosts := hostset.HealthyHosts()
	//logger := log.ByContext(context)

	if len(hosts) == 0 {
		//	logger.Debugf("Choose host failed, no health host found")
		return nil
	}

	hostIdx := l.randInstance.Intn(len(hosts))

	return hosts[hostIdx]
}

// TODO: more loadbalancers@boqin
type roundRobinLoadBalancer struct {
	loadbalancer
	// rrIndex for hostSet select
	rrIndexPriority uint32
	// rrIndex for host select
	rrIndex uint32
	lbMutex sync.RWMutex
}

func newRoundRobinLoadBalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &roundRobinLoadBalancer{
		loadbalancer: loadbalancer{
			prioritySet: prioritySet,
		},
	}
}

func (l *roundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	var selectedHostSet []types.Host

	hostSets := l.prioritySet.HostSetsByPriority()
	hostSetsNum := uint32(len(hostSets))
	curHostSet := hostSets[l.rrIndexPriority%hostSetsNum].HealthyHosts()

	if l.rrIndex >= uint32(len(curHostSet)) {
		l.lbMutex.Lock()
		l.rrIndexPriority = (l.rrIndexPriority + 1) % hostSetsNum
		l.rrIndex = 0
		l.lbMutex.Unlock()

		selectedHostSet = hostSets[l.rrIndexPriority].HealthyHosts()
	} else {
		selectedHostSet = curHostSet
	}

	if len(selectedHostSet) == 0 {
		//logger := log.ByContext(context)
		//logger.Debugf("Choose host in RoundRobin failed, no health host found")
		return nil
	}

	selectedHost := selectedHostSet[l.rrIndex%uint32(len(selectedHostSet))]
	atomic.AddUint32(&l.rrIndex, 1)

	return selectedHost
}

/*
SW (smoothWeightedRRLoadBalancer) is a struct that contains weighted items and provides methods to select a weighted item.
It is used for the smooth weighted round-robin balancing algorithm. This algorithm is implemented in Nginx:
https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35.
Algorithm is as follows: on each peer selection we increase current_weight
of each eligible peer by its weight, select peer with greatest current_weight
and reduce its current_weight by total number of weight points distributed
among peers.
In case of { 5, 1, 1 } weights this gives the following sequence of
current_weight's:
     a  b  c
     0  0  0  (initial state)

     5  1  1  (a selected)
    -2  1  1

     3  2  2  (a selected)
    -4  2  2

     1  3  3  (b selected)
     1 -4  3

     6 -3  4  (a selected)
    -1 -3  4

     4 -2  5  (c selected)
     4 -2 -2

     9 -1 -1  (a selected)
     2 -1 -1

     7  0  0  (a selected)
     0  0  0
*/

type smoothWeightedRRLoadBalancer struct {
	loadbalancer
	hostsWeighted map[string]*hostSmoothWeighted
}

type hostSmoothWeighted struct {
	weight          int
	currentWeight   int
	effectiveWeight int
}

func newSmoothWeightedRRLoadBalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	smoothWRRLoadBalancer := &smoothWeightedRRLoadBalancer{
		loadbalancer: loadbalancer{
			prioritySet: prioritySet,
		},
		hostsWeighted: make(map[string]*hostSmoothWeighted),
	}

	smoothWRRLoadBalancer.prioritySet.AddMemberUpdateCb(
		func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
			smoothWRRLoadBalancer.UpdateHost(priority, hostsAdded, hostsRemoved)
		},
	)

	hostSets := prioritySet.HostSetsByPriority()

	// iterate over all hosts to init host with Weighted
	for _, hostSet := range hostSets {
		for _, host := range hostSet.HealthyHosts() {
			smoothWRRLoadBalancer.hostsWeighted[host.AddressString()] = &hostSmoothWeighted{

				weight:          int(host.Weight()),
				effectiveWeight: int(host.Weight()),
			}
		}
	}

	return smoothWRRLoadBalancer
}

func (l *smoothWeightedRRLoadBalancer) UpdateHost(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
	// add host to hostWeighted
	for _, hostAdded := range hostsAdded {
		if _, ok := l.hostsWeighted[hostAdded.AddressString()]; !ok {
			// insert new health-host
			l.hostsWeighted[hostAdded.AddressString()] = &hostSmoothWeighted{
				weight:          int(hostAdded.Weight()),
				effectiveWeight: int(hostAdded.Weight()),
			}
		}
	}

	// remove host from hostsWeighted

	for _, hostRm := range hostsRemoved {
		delete(l.hostsWeighted, hostRm.AddressString())
	}
}

// smooth weighted round robin
// O(n), traverse over all hosts
// Insert new health host if not existed
func (l *smoothWeightedRRLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	totalWeight := 0
	var selectedHostWeighted *hostSmoothWeighted
	var selectedHost types.Host

	hostSets := l.prioritySet.HostSetsByPriority()
	for _, hosts := range hostSets {
		for _, host := range hosts.HealthyHosts() {

			if _, ok := l.hostsWeighted[host.AddressString()]; !ok {
				// insert new health-host in case UpdateHost not timely
				l.hostsWeighted[host.AddressString()] = &hostSmoothWeighted{
					weight:          int(host.Weight()),
					effectiveWeight: int(host.Weight()),
				}
			}

			hostW, _ := l.hostsWeighted[host.AddressString()]
			hostW.currentWeight += hostW.effectiveWeight
			totalWeight += hostW.effectiveWeight

			if hostW.effectiveWeight < hostW.weight {
				hostW.effectiveWeight++
			}

			if selectedHostWeighted == nil || hostW.currentWeight > selectedHostWeighted.currentWeight {
				selectedHostWeighted = hostW
				selectedHost = host
			}
		}
	}

	if selectedHostWeighted == nil {
		return nil
	}

	selectedHostWeighted.currentWeight -= totalWeight
	return selectedHost
}
