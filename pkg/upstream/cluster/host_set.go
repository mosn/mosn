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
	"strings"
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

func (hs *hostSet) Size() int {
	return len(hs.allHosts)
}

func (hs *hostSet) Get(i int) types.Host {
	if i < 0 {
		i = 0
	}
	if i >= len(hs.allHosts) {
		i = len(hs.allHosts) - 1
	}
	return hs.allHosts[i]
}

func (hs *hostSet) Range(f func(types.Host) bool) {
	for _, h := range hs.allHosts {
		if !f(h) {
			return
		}
	}
}

func (hs *hostSet) String() string {
	var sb strings.Builder
	sb.Grow(len(hs.allHosts) * 16)
	sb.WriteString("[")
	for i, h := range hs.allHosts {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(h.AddressString())
	}
	sb.WriteString("]")
	return sb.String()
}

func NewHostSet(hosts []types.Host) types.HostSet {
	// create distinct HostSet
	hs := &hostSet{}
	hs.setFinalHost(hosts)
	return hs
}
func NewNoDistinctHostSet(hosts []types.Host) types.HostSet {
	// create HostSet without distinct
	// In some scenarios, the hosts are always de duplicated.
	// such as do split hosts in load balancer
	// For performance reasons, there is no need to re duplicate them.
	// Therefore, the function of creating HostSet without distinct is provided
	return &hostSet{allHosts: hosts}
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
