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

package healthcheck

import (
	"sync"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// use a mock session Factory to generate a mock session
type mockSessionFactory struct {
}

func (f *mockSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	return &mockSession{host}
}

type mockSession struct {
	host types.Host
}

func (s *mockSession) CheckHealth() bool {
	if mh, ok := s.host.(*mockHost); ok {
		if mh.delay > 0 {
			time.Sleep(mh.delay)
		}
	}
	return s.host.Health()
}
func (s *mockSession) OnTimeout() {}

type mockCluster struct {
	types.Cluster
	hs *mockHostSet
}

func (c *mockCluster) HostSet() types.HostSet {
	return c.hs
}

func newMockHostSet(hosts []types.Host) types.HostSet {
	return &mockHostSet{hosts: hosts}
}

var _ types.HostSet = &mockHostSet{}

type mockHostSet struct {
	hosts []types.Host
}

func (hs *mockHostSet) Size() int {
	return len(hs.hosts)
}

func (hs *mockHostSet) Get(i int) types.Host {
	return hs.hosts[i]
}

func (hs *mockHostSet) Range(f func(types.Host) bool) {
	for _, h := range hs.hosts {
		if !f(h) {
			return
		}
	}
}

type mockHost struct {
	types.Host
	addr string
	flag uint64
	// mock status
	delay  time.Duration
	lock   sync.Mutex
	status bool
}

func (h *mockHost) SetHealth(health bool) {
	h.lock.Lock()
	h.status = health
	h.lock.Unlock()
}

func (h *mockHost) Health() bool {
	h.lock.Lock()
	health := h.status
	h.lock.Unlock()
	return health
}

func (h *mockHost) AddressString() string {
	return h.addr
}

func (h *mockHost) ClearHealthFlag(flag api.HealthFlag) {
	h.flag &= ^uint64(flag)
}

func (h *mockHost) ContainHealthFlag(flag api.HealthFlag) bool {
	return h.flag&uint64(flag) > 0
}

func (h *mockHost) SetHealthFlag(flag api.HealthFlag) {
	h.flag |= uint64(flag)
}
