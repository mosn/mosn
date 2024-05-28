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

	"mosn.io/mosn/pkg/types"
)

// leastActiveConnectionLoadBalancer choose the host with the least active connection
type leastActiveConnectionLoadBalancer struct {
	*EdfLoadBalancer
	choice               uint32
	activeConnectionBias float64
}

func newLeastActiveConnectionLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lb := &leastActiveConnectionLoadBalancer{}
	if info != nil && info.LbConfig() != nil {
		lb.choice = info.LbConfig().ChoiceCount
		lb.activeConnectionBias = info.LbConfig().ActiveRequestBias
	} else {
		lb.choice = defaultChoice
		lb.activeConnectionBias = defaultActiveRequestBias
	}
	lb.EdfLoadBalancer = newEdfLoadBalancer(info, hosts, lb.unweightChooseHost, lb.hostWeight)
	return lb
}

func (lb *leastActiveConnectionLoadBalancer) hostWeight(item WeightItem) float64 {
	host, ok := item.(types.Host)
	if !ok {
		return float64(item.Weight())
	}

	weight := fixHostWeight(float64(host.Weight()))

	biasedActiveConnection := math.Pow(float64(host.HostStats().UpstreamConnectionActive.Count())+1, lb.activeConnectionBias)

	return weight / biasedActiveConnection
}

// 1. This LB rely on HostStats, so make sure the host metrics statistic is enabled
// 2. Note that the same host in different clusters will share the same statistics
func (lb *leastActiveConnectionLoadBalancer) unweightChooseHost(context types.LoadBalancerContext) types.Host {
	hs := lb.hosts
	total := hs.Size()
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	var candidate types.Host
	// Choose `choice` times and return the best one
	// See The Power of Two Random Choices: A Survey of Techniques and Results
	//  http://www.eecs.harvard.edu/~michaelm/postscripts/handbook2001.pdf
	for cur := 0; cur < int(lb.choice); cur++ {

		randIdx := lb.rand.Intn(total)
		tempHost := hs.Get(randIdx)
		if candidate == nil {
			candidate = tempHost
			continue
		}
		if candidate.HostStats().UpstreamConnectionActive.Count() > tempHost.HostStats().UpstreamConnectionActive.Count() {
			candidate = tempHost
		}
	}
	return candidate

}
