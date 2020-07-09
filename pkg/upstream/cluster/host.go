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
	"sync/atomic"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// simpleHost is an implement of types.Host and types.HostInfo
type simpleHost struct {
	hostname      string
	addressString string
	clusterInfo   types.ClusterInfo
	stats         types.HostStats
	metaData      api.Metadata
	tlsDisable    bool
	weight        uint32
	healthFlags   *uint64
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
		healthFlags:   GetHealthFlagPointer(config.Address),
	}
}

// types.HostInfo Implement
func (sh *simpleHost) Hostname() string {
	return sh.hostname
}

func (sh *simpleHost) Metadata() api.Metadata {
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
	return IsSupportTLS() && !sh.tlsDisable && sh.clusterInfo.TLSMng().Enabled()
}

// types.Host Implement
func (sh *simpleHost) CreateConnection(context context.Context) types.CreateConnectionData {
	var tlsMng types.TLSContextManager
	if sh.SupportTLS() {
		tlsMng = sh.clusterInfo.TLSMng()
	}
	clientConn := network.NewClientConnection(nil, sh.clusterInfo.ConnectTimeout(), tlsMng, sh.Address(), nil)
	clientConn.SetBufferLimit(sh.clusterInfo.ConnBufferLimitBytes())

	return types.CreateConnectionData{
		Connection: clientConn,
		Host:       sh,
	}
}

func (sh *simpleHost) ClearHealthFlag(flag api.HealthFlag) {
	ClearHealthFlag(sh.healthFlags, flag)
}

func (sh *simpleHost) ContainHealthFlag(flag api.HealthFlag) bool {
	return atomic.LoadUint64(sh.healthFlags)&uint64(flag) > 0
}

func (sh *simpleHost) SetHealthFlag(flag api.HealthFlag) {
	SetHealthFlag(sh.healthFlags, flag)
}

func (sh *simpleHost) HealthFlag() api.HealthFlag {
	return api.HealthFlag(atomic.LoadUint64(sh.healthFlags))
}

func (sh *simpleHost) Health() bool {
	return atomic.LoadUint64(sh.healthFlags) == 0
}

// net.Addr reuse for same address, valid in simple type
// Update DNS cache using asynchronous mode
var AddrStore *utils.ExpiredMap = utils.NewExpiredMap(
	func(key interface{}) (interface{}, bool) {
		addr, err := net.ResolveTCPAddr("tcp", key.(string))
		if err == nil {
			return addr, true
		}
		return nil, false
	}, false)

func GetOrCreateAddr(addrstr string) net.Addr {

	if addr, _ := AddrStore.Get(addrstr); addr != nil {
		return addr.(net.Addr)
	}

	addr, err := net.ResolveTCPAddr("tcp", addrstr)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] resolve addr %s failed: %v", addrstr, err)
		return nil
	}

	if addr.String() != addrstr {
		// TODO support config or depends on DNS TTL for expire time
		// now set default expire time == 15 s, Means that after 15 seconds, the new request will trigger domain resolve.
		AddrStore.Set(addrstr, addr, 15*time.Second)
	} else {
		// if addrsstr isn't domain and don't set expire time
		AddrStore.Set(addrstr, addr, utils.NeverExpire)
	}

	return addr
}
