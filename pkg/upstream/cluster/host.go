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
	"net"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type hostSet struct {
	priority                uint32
	hosts                   []types.Host
	healthyHosts            []types.Host
	hostsPerLocality        [][]types.Host
	healthyHostsPerLocality [][]types.Host
	mux                     sync.RWMutex
	updateCallbacks         []types.MemberUpdateCallback
	metadata                v2.Metadata
}

func (hs *hostSet) Hosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hosts
}

func (hs *hostSet) HealthyHosts() []types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.healthyHosts
}

func (hs *hostSet) HostsPerLocality() [][]types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hostsPerLocality
}

func (hs *hostSet) HealthHostsPerLocality() [][]types.Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.healthyHostsPerLocality
}

func (hs *hostSet) UpdateHosts(hosts []types.Host, healthyHosts []types.Host, hostsPerLocality [][]types.Host,
	healthyHostsPerLocality [][]types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {

	// todo change mutex
	// modified because in updateCb(), there is lock condition
	hs.mux.Lock()
	hs.hosts = hosts
	hs.healthyHosts = healthyHosts
	hs.hostsPerLocality = hostsPerLocality
	hs.healthyHostsPerLocality = healthyHostsPerLocality
	hs.mux.Unlock()

	for _, updateCb := range hs.updateCallbacks {
		updateCb(hs.priority, hostsAdded, hostsRemoved)
	}
}

func (hs *hostSet) Priority() uint32 {
	return hs.priority
}

func (hs *hostSet) addMemberUpdateCb(cb types.MemberUpdateCallback) {
	hs.updateCallbacks = append(hs.updateCallbacks, cb)
}

// Host
type host struct {
	hostInfo
	weight uint32
	used   bool

	healthFlags uint64
}

// NewHost used to create types.Host
func NewHost(config v2.Host, clusterInfo types.ClusterInfo) types.Host {
	addr, _ := net.ResolveTCPAddr("tcp", config.Address)

	return &host{
		hostInfo: newHostInfo(addr, config, clusterInfo),
		weight:   config.Weight,
	}
}

func (h *host) CreateConnection(context context.Context) types.CreateConnectionData {
	logger := log.ByContext(context)
	var tlsMng types.TLSContextManager
	if !h.tlsDisable {
		tlsMng = h.clusterInfo.TLSMng()
	}

	clientConn := network.NewClientConnection(h.clusterInfo.SourceAddress(), tlsMng, h.address, nil, logger)
	clientConn.SetBufferLimit(h.clusterInfo.ConnBufferLimitBytes())

	return types.CreateConnectionData{
		Connection: clientConn,
		HostInfo:   &h.hostInfo,
	}
}

// health:0, unhealth:1
// set h.healthFlags = 0
// ^1 = 0
func (h *host) ClearHealthFlag(flag types.HealthFlag) {
	h.healthFlags &= ^uint64(flag)
}

// return 1, if h.healthFlags = 1
func (h *host) ContainHealthFlag(flag types.HealthFlag) bool {
	return h.healthFlags&uint64(flag) > 0
}

// set h.healthFlags = 1
func (h *host) SetHealthFlag(flag types.HealthFlag) {
	h.healthFlags |= uint64(flag)
}

// return 1 when h.healthFlags == 0
func (h *host) Health() bool {
	return h.healthFlags == 0
}

func (h *host) Weight() uint32 {
	return h.weight
}

func (h *host) SetWeight(weight uint32) {
	h.weight = weight
}

func (h *host) Used() bool {
	return h.used
}

func (h *host) SetUsed(used bool) {
	h.used = used
}

// HostInfo
type hostInfo struct {
	hostname       string
	address        net.Addr
	addressString  string
	canary         bool
	clusterInfo    types.ClusterInfo
	stats          types.HostStats
	metaData       types.RouteMetaData
	originMetaData v2.Metadata
	tlsDisable     bool
	config         v2.Host

	// TODO: locality, outlier, healthchecker
}

func newHostInfo(addr net.Addr, config v2.Host, clusterInfo types.ClusterInfo) hostInfo {
	var name string
	if clusterInfo != nil {
		name = clusterInfo.Name()
	}
	return hostInfo{
		address:        addr,
		addressString:  config.Address,
		hostname:       config.Hostname,
		clusterInfo:    clusterInfo,
		stats:          newHostStats(name, config.Address),
		metaData:       GenerateHostMetadata(config.MetaData),
		originMetaData: config.MetaData,
		tlsDisable:     config.TLSDisable,
		config:         config,
	}
}

func (hi *hostInfo) Hostname() string {
	return hi.hostname
}

func (hi *hostInfo) Canary() bool {
	return hi.canary
}

func (hi *hostInfo) Metadata() types.RouteMetaData {
	return hi.metaData
}

func (hi *hostInfo) OriginMetaData() v2.Metadata {
	return hi.originMetaData
}

func (hi *hostInfo) ClusterInfo() types.ClusterInfo {
	return hi.clusterInfo
}

func (hi *hostInfo) Address() net.Addr {
	return hi.address
}

func (hi *hostInfo) AddressString() string {
	return hi.addressString
}

func (hi *hostInfo) HostStats() types.HostStats {
	return hi.stats
}
func (hi *hostInfo) Config() v2.Host {
	return hi.config
}

// GenerateHostMetadata
// generate host's metadata in map[string]types.HashedValue type
func GenerateHostMetadata(metadata v2.Metadata) types.RouteMetaData {
	rm := make(map[string]types.HashedValue, 1)

	for k, v := range metadata {
		rm[k] = types.GenerateHashedValue(v)

	}

	return rm
}
