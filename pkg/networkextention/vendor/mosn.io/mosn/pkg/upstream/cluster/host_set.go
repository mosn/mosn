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
}

// Hosts do not needs lock, becasue it "immutable"
func (hs *hostSet) Hosts() []types.Host {
	return hs.allHosts
}

func (hs *hostSet) createSubset(predicate types.HostPredicate) types.HostSet {
	allHosts := hs.Hosts()
	var subHosts []types.Host
	for _, h := range allHosts {
		if predicate(h) {
			subHosts = append(subHosts, h)
		}
	}
	sub := &subHostSet{
		predicate: predicate,
		allHosts:  subHosts,
	}
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
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [host set] update host, final host total: %d", len(hs.allHosts))
		}
	})
}

// subHostSet is a types.Host makes by hostSet
// the member and healthy states is sync from hostSet that created it
// no callbacks can be added in subset hostset
type subHostSet struct {
	mux       sync.RWMutex
	allHosts  []types.Host
	predicate types.HostPredicate
}

// Hosts do not need lock, because it is "immutable"
func (sub *subHostSet) Hosts() []types.Host {
	return sub.allHosts
}
