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
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// hostSet is an implementation of types.HostSet
type hostSet struct {
	once     sync.Once
	mux      sync.RWMutex
	allHosts []types.Host
	// if refresh is true, the healthyHosts needs to refresh
	refresh       bool
	refreshNotify []func(host types.Host)
	healthyHosts  []types.Host
}

// Hosts do not needs lock, becasue it "immutable"
func (hs *hostSet) Hosts() []types.Host {
	return hs.allHosts
}

func (hs *hostSet) HealthyHosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()
	return hs.healthyHosts
}

// refresh notify do not needs lock, the createSubset will not be called in concurrency
// once the refreshNotify slice have been created, it will not be changed
func (hs *hostSet) addRefreshNotify(f func(host types.Host)) {
	hs.refreshNotify = append(hs.refreshNotify, f)
}

func (hs *hostSet) getRefreshNotify() []func(host types.Host) {
	return hs.refreshNotify
}

func (hs *hostSet) resetHealthyHosts() {
	healthyHosts := make([]types.Host, 0, len(hs.allHosts))
	for _, h := range hs.allHosts {
		if h.Health() {
			healthyHosts = append(healthyHosts, h)
		}
	}
	hs.mux.Lock()
	defer hs.mux.Unlock()
	hs.healthyHosts = healthyHosts

}

// refreshHealthHost resetHealthyHosts, and send a notify
func (hs *hostSet) refreshHealthHost(host types.Host) {
	hs.resetHealthyHosts()
	// send refresh notify
	refreshs := hs.getRefreshNotify()
	for _, refresh := range refreshs {
		refresh(host)
	}
}

func (hs *hostSet) createSubset(predicate types.HostPredicate) types.HostSet {
	allHosts := hs.Hosts()
	var subHosts []types.Host
	var healthyHosts []types.Host
	for _, h := range allHosts {
		if predicate(h) {
			subHosts = append(subHosts, h)
			if h.Health() {
				healthyHosts = append(healthyHosts, h)
			}
		}
	}
	sub := &subHostSet{
		predicate:    predicate,
		allHosts:     subHosts,
		healthyHosts: healthyHosts,
	}
	// register refresh notify
	hs.addRefreshNotify(sub.refresh)
	return sub
}

// setFinalHosts makes a slice copy, distinct host by address string
// can be set only once
func (hs *hostSet) setFinalHost(hosts []types.Host) {
	hs.once.Do(func() {
		// distinct hosts
		allHosts := make([]types.Host, 0, len(hosts))
		distinctHosts := map[string]struct{}{}
		for _, h := range hosts {
			addr := h.AddressString()
			if _, exists := distinctHosts[addr]; exists {
				continue
			}
			distinctHosts[addr] = struct{}{}
			allHosts = append(allHosts, h)
		}
		hs.allHosts = allHosts
		hs.resetHealthyHosts()
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [host set] update host, final host total: %d", len(hs.allHosts))
		}
	})
}

// subHostSet is a types.Host makes by hostSet
// the member and healthy states is sync from hostSet that created it
// no callbacks can be added in subset hostset
type subHostSet struct {
	mux          sync.RWMutex
	allHosts     []types.Host
	healthyHosts []types.Host
	predicate    types.HostPredicate
}

// Hosts do not need lock, because it is "immutable"
func (sub *subHostSet) Hosts() []types.Host {
	return sub.allHosts
}

func (sub *subHostSet) HealthyHosts() []types.Host {
	sub.mux.RLock()
	defer sub.mux.RUnlock()

	return sub.healthyHosts
}

func (sub *subHostSet) resetHealthyHosts() {
	healthyHosts := make([]types.Host, 0, len(sub.allHosts))
	for _, h := range sub.allHosts {
		if h.Health() {
			healthyHosts = append(healthyHosts, h)
		}
	}
	sub.mux.Lock()
	defer sub.mux.Unlock()
	sub.healthyHosts = healthyHosts
}

func (sub *subHostSet) refresh(host types.Host) {
	if !sub.predicate(host) {
		return
	}
	sub.resetHealthyHosts()
}
