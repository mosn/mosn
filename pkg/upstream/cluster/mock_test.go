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
	"context"
	"fmt"

	v2 "sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/network"
	"sofastack.io/sofa-mosn/pkg/router"
	"sofastack.io/sofa-mosn/pkg/types"
)

type mockHost struct {
	name       string
	addr       string
	meta       v2.Metadata
	healthFlag uint64
	types.Host
}

func (h *mockHost) Hostname() string {
	return h.name
}

func (h *mockHost) AddressString() string {
	return h.addr
}

func (h *mockHost) Metadata() v2.Metadata {
	return h.meta
}

func (h *mockHost) Health() bool {
	return h.healthFlag == 0
}

func (h *mockHost) ClearHealthFlag(flag types.HealthFlag) {
	h.healthFlag &= ^uint64(flag)
}

func (h *mockHost) SetHealthFlag(flag types.HealthFlag) {
	h.healthFlag |= uint64(flag)
}

func (h *mockHost) HealthFlag() types.HealthFlag {
	return types.HealthFlag(h.healthFlag)
}

type ipPool struct {
	idx int
	ips []string
}

func (pool *ipPool) Get() string {
	ip := pool.ips[pool.idx]
	pool.idx++
	return ip
}

func (pool *ipPool) MakeHosts(size int, meta v2.Metadata) []types.Host {
	hosts := make([]types.Host, size)
	for i := 0; i < size; i++ {
		host := &mockHost{
			addr: pool.Get(),
			meta: meta,
		}
		host.name = host.addr
		hosts[i] = host
	}
	return hosts
}

// makePool makes ${size} ips in a ipPool
func makePool(size int) *ipPool {
	var start int64 = 3221291264 // 192.1.1.0:80
	ips := make([]string, size)
	for i := 0; i < size; i++ {
		ip := start + int64(i)
		ips[i] = fmt.Sprintf("%d.%d.%d.%d:80", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
	}
	return &ipPool{
		ips: ips,
	}
}

type mockConnPool struct {
	h types.Host
	types.ConnectionPool
}

const mockProtocol = types.Protocol("mock")

func (p *mockConnPool) Protocol() types.Protocol {
	return mockProtocol
}

func (p *mockConnPool) CheckAndInit(ctx context.Context) bool {
	return true
}

func (p *mockConnPool) SupportTLS() bool {
	return p.h.SupportTLS()
}

func (p *mockConnPool) Shutdown() {
}

func init() {
	network.RegisterNewPoolFactory(mockProtocol, func(h types.Host) types.ConnectionPool {
		return &mockConnPool{
			h: h,
		}
	})
	types.RegisterConnPoolFactory(mockProtocol, true)
}

type mockLbContext struct {
	types.LoadBalancerContext
	mmc    types.MetadataMatchCriteria
	header types.HeaderMap
}

func newMockLbContext(m map[string]string) types.LoadBalancerContext {
	mmc := router.NewMetadataMatchCriteriaImpl(m)
	return &mockLbContext{
		mmc: mmc,
	}
}

func newMockLbContextWithHeader(m map[string]string, header types.HeaderMap) types.LoadBalancerContext {
	mmc := router.NewMetadataMatchCriteriaImpl(m)
	return &mockLbContext{
		mmc:    mmc,
		header: header,
	}
}

func (ctx *mockLbContext) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return ctx.mmc
}

func (ctx *mockLbContext) DownstreamHeaders() types.HeaderMap {
	return ctx.header
}

func (ctx *mockLbContext) DownstreamContext() context.Context {
	return nil
}
