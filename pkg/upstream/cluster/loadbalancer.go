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

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// Note: Random is the default lb
func NewLoadBalancer(lbType types.LoadBalancerType, prioritySet types.PrioritySet) types.LoadBalancer {
	switch lbType {
	case types.RoundRobin:
		return newRoundRobinLoadBalancer(prioritySet)
	default :
		return newRandomLoadbalancer(prioritySet)
	}

	return nil
}

type loadbalaner struct {
	prioritySet types.PrioritySet
}

// Random LoadBalancer
type randomLoadBalancer struct {
	loadbalaner
}

func newRandomLoadbalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &randomLoadBalancer{
		loadbalaner: loadbalaner{
			prioritySet: prioritySet,
		},
	}
}

func (l *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	hostSets := l.prioritySet.HostSetsByPriority()
	idx := rand.Intn(len(hostSets))
	hostset := hostSets[idx]

	hosts := hostset.HealthyHosts()
	//logger := log.ByContext(context)

	if len(hosts) == 0 {
		//	logger.Debugf("Choose host failed, no health host found")
		return nil
	}
	hostIdx := rand.Intn(len(hosts))
	return hosts[hostIdx]
}

// TODO: more loadbalancers@boqin
type roundRobinLoadBalancer struct {
	loadbalaner
	// rrIndex for hostSet select
	rrIndexPriority uint32
	// rrIndex for host select
	rrIndex uint32
}

func newRoundRobinLoadBalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &roundRobinLoadBalancer{
		loadbalaner: loadbalaner{
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
		l.rrIndexPriority = (l.rrIndexPriority + 1) % hostSetsNum
		l.rrIndex = 0
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
	l.rrIndex++

	return selectedHost
}
