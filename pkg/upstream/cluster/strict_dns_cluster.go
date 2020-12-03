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
 * distributed under the License is distributed on an "AS IS" BASIS, * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"math"
	"net"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

func init() {
	RegisterClusterType(v2.STRICT_DNS_CLUSTER, newStrictDnsCluster)
}

type strictDnsCluster struct {
	*simpleCluster
	dnsResolver     *network.DnsResolver
	dnsLookupFamily v2.DnsLookupFamily
	respectDnsTTL   bool
	resolveTargets  []*ResolveTarget
	dnsRefreshRate  time.Duration
	mutex           sync.Mutex
	version         uint64
}

var DefaultRefreshTimeout time.Duration = 20 * time.Second
var DefaultRefreshInterval time.Duration = 5 * time.Second

type ResolveTarget struct {
	config           *v2.Host
	dnsAddress       string
	port             string
	strictDnsCluster *strictDnsCluster
	resolveTimer     *utils.Timer
	resolveTimeout   *utils.Timer
	refreshTimeout   time.Duration
	dnsRefreshRate   chan time.Duration
	stop             chan struct{}
	timeout          chan bool
	resolveCount     uint32
	hosts            []types.Host
	version          uint64
}

func newStrictDnsCluster(clusterConfig v2.Cluster) types.Cluster {

	cluster := &strictDnsCluster{
		simpleCluster:   newSimpleCluster(clusterConfig).(*simpleCluster),
		dnsLookupFamily: clusterConfig.DnsLookupFamily,
		respectDnsTTL:   clusterConfig.RespectDnsTTL,
		resolveTargets:  []*ResolveTarget{},
		mutex:           sync.Mutex{},
	}

	// set dnsRefreshRate
	if clusterConfig.DnsRefreshRate != nil {
		cluster.dnsRefreshRate = clusterConfig.DnsRefreshRate.Duration
	}

	// set resolve server
	if clusterConfig.DnsResolverConfig.Servers != nil {
		cluster.dnsResolver = network.NewDnsResolver(&clusterConfig.DnsResolverConfig)
	} else {
		cluster.dnsResolver = network.NewDnsResolverFromFile(clusterConfig.DnsResolverFile, clusterConfig.DnsResolverPort)
	}

	return cluster
}

// supported formats including {aaa.com:80, aaa.com}
func getHostPortFromAddr(addr string) (string, string) {
	s := strings.Split(addr, ":")
	if len(s) == 1 {
		return s[0], ""
	} else {
		return s[0], s[1]
	}
}

func (sdc *strictDnsCluster) UpdateHosts(newHosts []types.Host) {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()
	sdc.StopResolve()
	atomic.AddUint64(&sdc.version, 1)

	// before resolve, init hosts to origin unresolved address
	sdc.simpleCluster.UpdateHosts(newHosts)

	// update resolve targets and start new resolve task
	var rts []*ResolveTarget
	for _, host := range newHosts {
		addr, port := getHostPortFromAddr(host.AddressString())
		if addr == "" {
			if log.DefaultLogger.GetLogLevel() >= log.ERROR {
				log.DefaultLogger.Errorf("[upstream] [strict_dns_cluster] config address format error: %s", host.AddressString())
			}
			continue
		}
		// default port: 80
		if port == "" {
			port = "80"
		}
		config := host.Config()
		rt := &ResolveTarget{
			dnsAddress:       addr,
			port:             port,
			config:           &config,
			refreshTimeout:   DefaultRefreshTimeout,
			dnsRefreshRate:   make(chan time.Duration),
			stop:             make(chan struct{}),
			timeout:          make(chan bool),
			strictDnsCluster: sdc,
			hosts:            []types.Host{host},
			version:          sdc.version,
		}

		rts = append(rts, rt)
		// if address is already an ip, skip dns resolution
		if net.ParseIP(rt.dnsAddress) != nil {
			continue
		}
		utils.GoWithRecover(func() {
			rt.StartResolve()
		}, nil)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [strict dns cluster] create a resolver for address: %s", rt.dnsAddress)
		}
	}
	sdc.resolveTargets = rts
}

func (sdc *strictDnsCluster) StopResolve() {
	for _, rt := range sdc.resolveTargets {
		rt.StopResolve()
	}
}

// calculate next resolve interval
// 1. if respectDnsTTL configured, and minTtl > 0, use minTtl+1
// 2. if dnsRefreshRate configured, use dnsRefreshRate
// 3. use DefaultRefreshInterval(5s)
func (sdc *strictDnsCluster) calculateNextResolveInterval(minTtl time.Duration) time.Duration {
	dnsRefreshRate := sdc.dnsRefreshRate
	if sdc.respectDnsTTL && minTtl > 0 {
		// increase minTtl by 1 in case we will get dns response with ttl 0
		dnsRefreshRate = minTtl + time.Second
	} else if sdc.dnsRefreshRate == 0 {
		dnsRefreshRate = DefaultRefreshInterval
	}
	return dnsRefreshRate
}

// just check address string, hostname and metadata
func hostEqual(hosts1, hosts2 *[]types.Host) bool {
	hosts := map[string]types.Host{}
	if len(*hosts1) != len(*hosts2) {
		return false
	}
	for _, host := range *hosts1 {
		hosts[host.AddressString()] = host
	}
	for _, host := range *hosts2 {
		if h, exists := hosts[host.AddressString()]; exists {
			if h.Hostname() == host.Hostname() && reflect.DeepEqual(h.Metadata(), host.Metadata()) {
				continue
			}
			return false
		}
		return false
	}
	return true
}

// get all hosts in current rt and other rt, do updating if hosts changed
func (sdc *strictDnsCluster) updateDynamicHosts(newHosts []types.Host, rt *ResolveTarget) {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()
	// if sdc updated by UpdateHosts, skip updating
	ver := atomic.LoadUint64(&sdc.version)
	if ver != rt.version {
		return
	}

	// update current hosts in rt
	rt.hosts = newHosts

	var allHosts []types.Host
	// collect all hosts in resolve targets
	for _, r := range sdc.resolveTargets {
		for _, h := range r.hosts {
			allHosts = append(allHosts, h)
		}
	}

	// compare current hosts with allHosts
	hostNotChanged := hostEqual(&allHosts, &sdc.hostSet.allHosts)
	if !hostNotChanged {
		for _, h := range newHosts {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Infof("[upstream] [strict dns cluster] resolve dns new address:%s", h.AddressString())
			}
		}
		sdc.simpleCluster.UpdateHosts(newHosts)
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [strict dns cluster] resolve dns result updated, cluster_name:%s, address:%s", sdc.simpleCluster.info.name, rt.dnsAddress)
		}
	}
}

func (rt *ResolveTarget) StopResolve() {
	close(rt.stop)
}

func (rt *ResolveTarget) StartResolve() {
	defer func() {
		if r := recover(); r != nil {
			if log.DefaultLogger.GetLogLevel() >= log.ERROR {
				log.DefaultLogger.Errorf("[upstream] [strict_dns_cluster] [resolver] panic %v\n%s", r, string(debug.Stack()))
			}
		}
		rt.resolveTimer.Stop()
		rt.resolveTimeout.Stop()
	}()

	// start resolve now
	rt.resolveTimer = utils.NewTimer(0, rt.OnResolve)
	for {
		select {
		case <-rt.stop:
			rt.resolveTimeout.Stop()
			rt.resolveTimer.Stop()
		default:
			select {
			case <-rt.stop:
				if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
					log.DefaultLogger.Debugf("[upstream] [strict dns cluster] stop resolve dns timer, address:%s, ttl:%d", rt.dnsAddress)
				}
				rt.resolveTimeout.Stop()
				rt.resolveTimer.Stop()
			case <-rt.timeout:
				rt.resolveTimer.Stop()
				// if timeout, start a new timer
				rt.resolveTimer = utils.NewTimer(time.Second, rt.OnResolve)
				if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
					log.DefaultLogger.Debugf("[upstream] [strict dns cluster] timeout received when resolve dns address :%s", rt.dnsAddress)
				}
			case ttl := <-rt.dnsRefreshRate:
				rt.resolveTimeout.Stop()
				// next resolve timer
				rt.resolveTimer = utils.NewTimer(ttl, rt.OnResolve)
				if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
					log.DefaultLogger.Debugf("[upstream] [strict dns cluster] start next resolve dns timer, address:%s, ttl:%d", rt.dnsAddress, ttl)
				}
			}
		}
	}
}

func (rt *ResolveTarget) OnTimeout() {
	rt.timeout <- true
}

func (rt *ResolveTarget) OnResolve() {
	rt.resolveTimeout.Stop()
	rt.resolveTimeout = utils.NewTimer(rt.refreshTimeout, rt.OnTimeout)
	sdc := rt.strictDnsCluster
	dnsResponse := sdc.dnsResolver.DnsResolve(rt.dnsAddress, sdc.dnsLookupFamily)
	if dnsResponse == nil {
		rt.dnsRefreshRate <- sdc.calculateNextResolveInterval(0)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [strict dns cluster] resolve failed and start a new task")
		}
		return
	}
	// record min ttl value in responses
	minTtl := math.MaxInt16 * time.Second
	var hosts []types.Host
	stat := newHostStats(sdc.info.Name(), rt.config.Address)
	for _, rsp := range *dnsResponse {
		if rsp.Ttl < minTtl {
			minTtl = rsp.Ttl
		}
		var newAddr string
		newAddr = rsp.Address + ":" + rt.port
		host := &simpleHost{
			hostname:      rt.config.Hostname,
			addressString: newAddr,
			stats:         stat,
			metaData:      rt.config.MetaData,
			tlsDisable:    rt.config.TLSDisable,
			weight:        rt.config.Weight,
			healthFlags:   GetHealthFlagPointer(newAddr),
		}
		host.clusterInfo.Store(sdc.info)
		hosts = append(hosts, host)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [strict dns cluster] resolve dns result, address:%s, addr:%s, ttl:%.3f", rt.dnsAddress, newAddr, rsp.Ttl.Seconds())
		}
	}
	sdc.updateDynamicHosts(hosts, rt)

	rt.dnsRefreshRate <- sdc.calculateNextResolveInterval(minTtl)
}
