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
	"net"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type dynamicClusterBase struct {
	cluster
}

func (dc *dynamicClusterBase) updateDynamicHostList(newHosts []types.Host, currentHosts []types.Host) (
	changed bool, finalHosts []types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {
	hostAddrs := make(map[string]bool)

	// N^2 loop, works for small and steady hosts
	for _, nh := range newHosts {
		nhAddr := nh.AddressString()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}

		hostAddrs[nhAddr] = true

		found := false
		for i := 0; i < len(currentHosts); {
			curNh := currentHosts[i]

			if nh.AddressString() == curNh.AddressString() {
				curNh.SetWeight(nh.Weight())
				finalHosts = append(finalHosts, curNh)
				currentHosts = append(currentHosts[:i], currentHosts[i+1:]...)
				found = true
			} else {
				i++
			}
		}

		if !found {
			finalHosts = append(finalHosts, nh)
			hostsAdded = append(hostsAdded, nh)
		}
	}

	if len(currentHosts) > 0 {
		hostsRemoved = currentHosts
	}

	if len(hostsAdded) > 0 || len(hostsRemoved) > 0 {
		changed = true
	} else {
		changed = false
	}

	return changed, finalHosts, hostsAdded, hostsRemoved
}

// SimpleCluster
type simpleInMemCluster struct {
	dynamicClusterBase

	hosts []types.Host
}

func newSimpleInMemCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool) *simpleInMemCluster {
	cluster := newCluster(clusterConfig, sourceAddr, addedViaAPI, nil)

	return &simpleInMemCluster{
		dynamicClusterBase: dynamicClusterBase{
			cluster: cluster,
		},
	}
}

func (sc *simpleInMemCluster) UpdateHosts(newHosts []types.Host) {
	sc.mux.Lock()
	defer sc.mux.Unlock()

	var curHosts = make([]types.Host, len(sc.hosts))

	copy(curHosts, sc.hosts)
	changed, finalHosts, hostsAdded, hostsRemoved := sc.updateDynamicHostList(newHosts, curHosts)

	log.DefaultLogger.Debugf("update host changed %t", changed)

	if changed {
		sc.hosts = finalHosts
		// todo: need to consider how to update healthyHost
		// Note: currently, we only use priority 0
		sc.prioritySet.GetOrCreateHostSet(0).UpdateHosts(sc.hosts,
			sc.hosts, nil, nil, hostsAdded, hostsRemoved)

		if sc.healthChecker != nil {
			sc.healthChecker.OnClusterMemberUpdate(hostsAdded, hostsRemoved)
		}

	}

	if len(sc.hosts) == 0 {
		log.DefaultLogger.Debugf(" after update final host is []")
	}

	for i, f := range sc.hosts {
		log.DefaultLogger.Debugf("after update final host index = %d, address = %s,", i, f.AddressString())
	}
}
