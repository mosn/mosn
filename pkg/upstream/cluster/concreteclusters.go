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
	"sort"
	"time"

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/types"
)

type dynamicClusterBase struct {
	*cluster
}

func (dc *dynamicClusterBase) updateDynamicHostList(newHosts []types.Host, currentHosts []types.Host) (
	changed bool, finalHosts []types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {

	sortedCurrentHosts := types.SortedHosts(currentHosts)
	sort.Sort(sortedCurrentHosts)
	hostAddrs := make(map[string]bool)

	for _, nh := range newHosts {
		nhAddr := nh.AddressString()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}

		hostAddrs[nhAddr] = true

		i := sort.Search(sortedCurrentHosts.Len(), func(i int) bool {
			return sortedCurrentHosts[i].AddressString() >= nhAddr
		})

		found := false

		if i < sortedCurrentHosts.Len() && sortedCurrentHosts[i].AddressString() == nhAddr {
			curNh := sortedCurrentHosts[i]
			curNh.SetWeight(nh.Weight())
			finalHosts = append(finalHosts, curNh)
			sortedCurrentHosts = append(sortedCurrentHosts[:i], sortedCurrentHosts[i+1:]...)
			found = true
		}
		if !found {
			finalHosts = append(finalHosts, nh)
			hostsAdded = append(hostsAdded, nh)
		}
	}

	if len(sortedCurrentHosts) > 0 {
		hostsRemoved = sortedCurrentHosts
	}

	if len(hostsAdded) > 0 || len(hostsRemoved) > 0 {
		changed = true
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

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[upstream] [simple cluster] update host changed %t", changed)
	}

	if changed {
		sc.hosts = finalHosts
		// Note: currently, we only use priority 0
		// we should choose the healthy host, default is healthy
		healthyHosts := make([]types.Host, 0, len(finalHosts))
		for _, h := range finalHosts {
			if h.Health() {
				healthyHosts = append(healthyHosts, h)
			}
		}
		sc.prioritySet.GetOrCreateHostSet(0).UpdateHosts(sc.hosts, healthyHosts, hostsAdded, hostsRemoved)

		if sc.healthChecker != nil {
			sc.healthChecker.OnClusterMemberUpdate(hostsAdded, hostsRemoved)
		}
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [simple cluster] update host, final host total: %d", len(finalHosts))
		}
	}
}

type originalDstCluster struct {
	dynamicClusterBase
	hosts  []types.Host
	stopCh chan int
}

func newOriginalDstCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool) *originalDstCluster {
	cluster := newCluster(clusterConfig, sourceAddr, addedViaAPI, nil)

	originalDstCluster := originalDstCluster{
		dynamicClusterBase: dynamicClusterBase{
			cluster: cluster,
		},
		stopCh: make(chan int, 1),
	}

	go func() {
		interval := originalDstCluster.info.cleanupInterval
		if interval < time.Second {
			interval = time.Second
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				originalDstCluster.cleanup()
			case <-originalDstCluster.stopCh:
				return
			}
		}
	}()

	return &originalDstCluster
}

func (sc *originalDstCluster) UpdateHosts(newHosts []types.Host) {
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
			sc.hosts, hostsAdded, hostsRemoved)

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

func (sc *originalDstCluster) cleanup() {
	newHost := make([]types.Host, 0)
	hostToRemove := make([]types.Host, 0)
	hostSet := sc.prioritySet.GetOrCreateHostSet(0)
	for _, host := range hostSet.Hosts() {
		if host.Used() {
			host.SetUsed(false)
			newHost = append(newHost, host)
		} else {
			hostToRemove = append(hostToRemove, host)
		}
	}
	hostSet.UpdateHosts(newHost, newHost, []types.Host{}, hostToRemove)
}

func (sc *originalDstCluster) stop() {
	sc.stopCh <- 1
}