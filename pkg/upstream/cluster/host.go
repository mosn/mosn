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

	v2 "sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/network"
	"sofastack.io/sofa-mosn/pkg/types"
)

// simpleHost is an implement of types.Host and types.HostInfo
type simpleHost struct {
	hostname      string
	addressString string
	clusterInfo   types.ClusterInfo
	stats         types.HostStats
	metaData      v2.Metadata
	tlsDisable    bool
	weight        uint32
	healthFlags   uint64
}

func NewSimpleHost(config v2.Host, clusterInfo types.ClusterInfo) types.Host {
	// clusterInfo should not be nil
	// pre resolve address
	GetOrCreateAddr(config.Address)
	return &simpleHost{
		hostname:      config.Hostname,
		addressString: config.Address,
		clusterInfo:   clusterInfo,
		stats:         newHostStats(clusterInfo.Name(), config.Address),
		metaData:      config.MetaData,
		tlsDisable:    config.TLSDisable,
		weight:        config.Weight,
	}
}

// types.HostInfo Implement
func (sh *simpleHost) Hostname() string {
	return sh.hostname
}

func (sh *simpleHost) Metadata() v2.Metadata {
	return sh.metaData
}

func (sh *simpleHost) ClusterInfo() types.ClusterInfo {
	return sh.clusterInfo
}

func (sh *simpleHost) Address() net.Addr {
	return GetOrCreateAddr(sh.addressString)
}

func (sh *simpleHost) AddressString() string {
	return sh.addressString
}

func (sh *simpleHost) HostStats() types.HostStats {
	return sh.stats
}

func (sh *simpleHost) Weight() uint32 {
	return sh.weight
}

func (sh *simpleHost) Config() v2.Host {
	return v2.Host{
		HostConfig: v2.HostConfig{
			Address:    sh.addressString,
			Hostname:   sh.hostname,
			TLSDisable: sh.tlsDisable,
			Weight:     sh.weight,
		},
		MetaData: sh.metaData,
	}
}

func (sh *simpleHost) SupportTLS() bool {
	return !sh.tlsDisable && sh.clusterInfo.TLSMng().Enabled()
}

// types.Host Implement
func (sh *simpleHost) CreateConnection(context context.Context) types.CreateConnectionData {
	var tlsMng types.TLSContextManager
	if !sh.tlsDisable {
		tlsMng = sh.clusterInfo.TLSMng()
	}
	clientConn := network.NewClientConnection(nil, sh.clusterInfo.ConnectTimeout(), tlsMng, sh.Address(), nil)
	clientConn.SetBufferLimit(sh.clusterInfo.ConnBufferLimitBytes())

	return types.CreateConnectionData{
		Connection: clientConn,
		HostInfo:   sh,
	}
}

func (sh *simpleHost) ClearHealthFlag(flag types.HealthFlag) {
	sh.healthFlags &= ^uint64(flag)
}

func (sh *simpleHost) ContainHealthFlag(flag types.HealthFlag) bool {
	return sh.healthFlags&uint64(flag) > 0
}

func (sh *simpleHost) SetHealthFlag(flag types.HealthFlag) {
	sh.healthFlags |= uint64(flag)
}

func (sh *simpleHost) HealthFlag() types.HealthFlag {
	return types.HealthFlag(sh.healthFlags)
}

func (sh *simpleHost) Health() bool {
	return sh.healthFlags == 0
}

// net.Addr reuse for same address, valid in simple type
var AddrStore sync.Map

func GetOrCreateAddr(addrstr string) net.Addr {
	if addr, ok := AddrStore.Load(addrstr); ok {
		return addr.(net.Addr)
	}
	addr, err := net.ResolveTCPAddr("tcp", addrstr)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] resolve addr %s failed: %v", addrstr, err)
		return nil
	}
	AddrStore.Store(addrstr, addr)
	return addr
}
