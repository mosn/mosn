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
	"sort"
	"sync"

	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/types"
)

// hostSet is an implementation of types.HostSet
type hostSet struct {
	mux       sync.RWMutex
	allHosts  []types.Host
	callbacks []types.MemberUpdateCallback
	// if refresh is true, the healthyHosts needs to refresh
	refresh       bool
	refreshNotify []func(host types.Host)
	healthyHosts  []types.Host
}

func (hs *hostSet) Hosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.allHosts
}

func (hs *hostSet) HealthyHosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()
	return hs.healthyHosts
}

func (hs *hostSet) AdddMemberUpdateCb(cb types.MemberUpdateCallback) {
	hs.mux.Lock()
	defer hs.mux.Unlock()
	hs.callbacks = append(hs.callbacks, cb)
}

func (hs *hostSet) addRefreshNotify(f func(host types.Host)) {
	hs.mux.Lock()
	defer hs.mux.Unlock()
	hs.refreshNotify = append(hs.refreshNotify, f)
}

// avoid loop calls
func (hs *hostSet) getMemberUpdateCb() []types.MemberUpdateCallback {
	hs.mux.RLock()
	defer hs.mux.RUnlock()
	return hs.callbacks[:len(hs.callbacks)]
}

func (hs *hostSet) getRefreshNotify() []func(host types.Host) {
	hs.mux.RLock()
	defer hs.mux.RUnlock()
	return hs.refreshNotify[:len(hs.refreshNotify)]
}

// refreshHealthHosts change the healthy hosts
func (hs *hostSet) refreshHealthHosts(host types.Host) {
	// change healthy hosts
	func() {
		hs.mux.Lock()
		defer hs.mux.Unlock()
		if host.Health() {
			hs.healthyHosts = append(hs.healthyHosts, host)
		} else {
			for idx, hh := range hs.healthyHosts {
				if hh.AddressString() == host.AddressString() {
					hs.healthyHosts = append(hs.healthyHosts[:idx], hs.healthyHosts[idx+1:]...)
					return
				}
			}
		}
	}()
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
	// regiter callback
	hs.AdddMemberUpdateCb(sub.memberUpdate)
	// register refresh notify
	hs.addRefreshNotify(sub.refresh)
	return sub
}

func (hs *hostSet) setFinalHost(hosts []types.Host) {
	// set final hosts
	hs.mux.Lock()
	defer hs.mux.Unlock()
	hs.allHosts = hosts
	// member changed, needs to refresh the host healthy states, default is health
	healthyHosts := hs.healthyHosts[:0]
	for _, h := range hs.allHosts {
		if h.Health() {
			healthyHosts = append(healthyHosts, h)
		}
	}
	hs.healthyHosts = healthyHosts
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[upstream] [host set] update host, final host total: %d", len(hs.allHosts))
	}
}

func (hs *hostSet) UpdateHosts(newHosts []types.Host) {
	allHosts := hs.Hosts()
	var curHosts = make([]types.Host, len(allHosts))
	copy(curHosts, allHosts)

	diff := hs.hostsCompare(newHosts, curHosts)

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[upstream] [host set] update host changed %t", diff.changed)
	}

	if diff.changed {
		hs.setFinalHost(diff.finalHosts)
		// callback
		// 1. active healthy check
		// 2. subset host set
		// 3. others
		callbacks := hs.getMemberUpdateCb()
		for _, cb := range callbacks {
			cb(diff.hostsAdded, diff.hostsRemoved)
		}
	}
}

// RemoveHosts is a optimize for fast remove hosts in host set
func (hs *hostSet) RemoveHosts(addrs []string) {
	allHosts := hs.Hosts()
	var curHosts = make([]types.Host, len(allHosts))
	copy(curHosts, allHosts)
	sortedCurrentHosts := types.SortedHosts(curHosts)
	sort.Sort(sortedCurrentHosts)
	hostsRemoved := make([]types.Host, 0, len(addrs))
	for _, addr := range addrs {
		i := sort.Search(sortedCurrentHosts.Len(), func(i int) bool {
			return sortedCurrentHosts[i].AddressString() >= addr
		})
		// found it, delete it
		if i < sortedCurrentHosts.Len() && sortedCurrentHosts[i].AddressString() == addr {
			hostsRemoved = append(hostsRemoved, sortedCurrentHosts[i])
			sortedCurrentHosts = append(sortedCurrentHosts[:i], sortedCurrentHosts[i+1:]...)
		}
	}
	hs.setFinalHost(sortedCurrentHosts)
	// callback
	callbacks := hs.getMemberUpdateCb()
	for _, cb := range callbacks {
		cb(nil, hostsRemoved)
	}
}

type hostDiff struct {
	changed      bool
	finalHosts   []types.Host
	hostsAdded   []types.Host
	hostsRemoved []types.Host
}

func (hs *hostSet) hostsCompare(newHosts, currentHosts []types.Host) *hostDiff {
	diff := &hostDiff{}
	sortedCurrentHosts := types.SortedHosts(currentHosts)
	sort.Sort(sortedCurrentHosts)
	hostAddrs := make(map[string]bool)
	for _, nh := range newHosts {
		nhAddr := nh.AddressString()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}
		i := sort.Search(sortedCurrentHosts.Len(), func(i int) bool {
			return sortedCurrentHosts[i].AddressString() >= nhAddr
		})

		if i < sortedCurrentHosts.Len() && sortedCurrentHosts[i].AddressString() == nhAddr {
			// update host
			// should use new host,but keep the health flag
			curHost := sortedCurrentHosts[i]
			nh.SetHealthFlag(curHost.HealthFlag())
			diff.finalHosts = append(diff.finalHosts, nh)
			sortedCurrentHosts = append(sortedCurrentHosts[:i], sortedCurrentHosts[i+1:]...)
		} else {
			// add new host
			diff.finalHosts = append(diff.finalHosts, nh)
			diff.hostsAdded = append(diff.hostsAdded, nh)
		}
	}
	// the hosts not new and not updated, delete it
	if len(sortedCurrentHosts) > 0 {
		diff.hostsRemoved = sortedCurrentHosts
	}

	if len(diff.hostsAdded) > 0 || len(diff.hostsRemoved) > 0 {
		diff.changed = true
		sort.Sort(types.SortedHosts(diff.finalHosts))
	}

	return diff
}

// subHostSet is a types.Host makes by hostSet
// the member and healthy states is sync from hostSet that created it
// no callbacks can be added in subset hostset
type subHostSet struct {
	mux          sync.RWMutex
	allHosts     []types.Host
	healthyHosts []types.Host
	predicate    types.HostPredicate
	callbacks    []types.MemberUpdateCallback
}

func (sub *subHostSet) Hosts() []types.Host {
	sub.mux.RLock()
	defer sub.mux.RUnlock()

	return sub.allHosts
}

func (sub *subHostSet) HealthyHosts() []types.Host {
	sub.mux.RLock()
	defer sub.mux.RUnlock()

	return sub.healthyHosts
}

func (sub *subHostSet) AdddMemberUpdateCb(cb types.MemberUpdateCallback) {
	sub.mux.Lock()
	defer sub.mux.Unlock()

	sub.callbacks = append(sub.callbacks, cb)
}

func (sub *subHostSet) getMemberUpdateCb() []types.MemberUpdateCallback {
	sub.mux.RLock()
	defer sub.mux.RUnlock()
	return sub.callbacks[:len(sub.callbacks)]
}

func (sub *subHostSet) UpdateHosts(newHosts []types.Host) {
	// do nothing
	// sub host set's member comes from host set
}

func (sub *subHostSet) RemoveHosts(addrs []string) {
	// do nothing
	// sub host set's member comes from host set
}

func (sub *subHostSet) refresh(host types.Host) {
	if !sub.predicate(host) {
		return
	}
	sub.mux.Lock()
	defer sub.mux.Unlock()
	// change healthy hosts
	if host.Health() {
		sub.healthyHosts = append(sub.healthyHosts, host)
	} else {
		for idx, hh := range sub.healthyHosts {
			if hh.AddressString() == host.AddressString() {
				sub.healthyHosts = append(sub.healthyHosts[:idx], sub.healthyHosts[idx+1:]...)
				return
			}
		}
	}
}

func (sub *subHostSet) memberUpdate(hostsAdded []types.Host, hostsRemoved []types.Host) {
	adds := make([]types.Host, 0, len(hostsAdded))
	removes := make([]types.Host, 0, len(hostsRemoved))
	// filter matched hosts
	for _, host := range hostsAdded {
		if sub.predicate(host) {
			adds = append(adds, host)
		}
	}
	for _, host := range hostsRemoved {
		if sub.predicate(host) {
			removes = append(removes, host)
		}
	}
	//
	allHosts := sub.getFinalHosts(adds, removes)
	func() {
		sub.mux.Lock()
		defer sub.mux.Unlock()
		sub.allHosts = allHosts
		healthyHosts := sub.healthyHosts[:0]
		for _, host := range sub.allHosts {
			if host.Health() {
				healthyHosts = append(healthyHosts, host)
			}
		}
		sub.healthyHosts = healthyHosts
	}()
	callbacks := sub.getMemberUpdateCb()
	for _, cb := range callbacks {
		cb(hostsAdded, hostsRemoved)
	}
}

func (sub *subHostSet) getFinalHosts(hostsAdded []types.Host, hostsRemoved []types.Host) []types.Host {
	sortedHosts := types.SortedHosts(sub.allHosts)
	sort.Sort(sortedHosts)

	originLen := sortedHosts.Len()
	for _, host := range hostsAdded {
		addr := host.AddressString()
		i := sort.Search(originLen, func(i int) bool {
			return sortedHosts[i].AddressString() >= addr
		})
		if !(i < originLen && sortedHosts[i].AddressString() == addr) {
			sortedHosts = append(sortedHosts, host)
		}
	}

	sort.Sort(sortedHosts)

	for _, host := range hostsRemoved {
		addr := host.AddressString()
		i := sort.Search(sortedHosts.Len(), func(i int) bool {
			return sortedHosts[i].AddressString() >= addr
		})
		// found
		if i < sortedHosts.Len() && sortedHosts[i].AddressString() == addr {
			sortedHosts = append(sortedHosts[:i], sortedHosts[i+1:]...)
		}
	}

	return sortedHosts

}
