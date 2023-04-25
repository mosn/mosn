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

package configmanager

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/automaxprocs/maxprocs"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

type ContentKey string

// ParsedCallback is an
// alias for closure func(data interface{}, endParsing bool) error
type ParsedCallback func(data interface{}, endParsing bool) error

// basic config
var configParsedCBMaps = make(map[ContentKey][]ParsedCallback)

// Group of ContentKey
// notes: configcontentkey equals to the key of config file
const (
	ParseCallbackKeyCluster   ContentKey = "clusters"
	ParseCallbackKeyProcessor ContentKey = "processor"
)

// RegisterConfigParsedListener
// used to register ParsedCallback
func RegisterConfigParsedListener(key ContentKey, cb ParsedCallback) {
	if cbs, ok := configParsedCBMaps[key]; ok {
		cbs = append(cbs, cb)
		// append maybe change the slice, should be assigned again
		configParsedCBMaps[key] = cbs
	} else {
		log.StartLogger.Infof("[config] %s added to configParsedCBMaps", key)
		cpc := []ParsedCallback{cb}
		configParsedCBMaps[key] = cpc
	}
}

// ParseClusterConfig parses config data to api data, verify whether the config is valid
func ParseClusterConfig(clusters []v2.Cluster) ([]v2.Cluster, map[string][]v2.Host) {
	if len(clusters) == 0 {
		log.StartLogger.Warnf("[config] [parse cluster] No Cluster provided in cluster config")
	}
	var pClusters []v2.Cluster
	clusterV2Map := make(map[string][]v2.Host)
	for _, c := range clusters {
		if c.Name == "" {
			log.StartLogger.Fatalf("[config] [parse cluster] name is required in cluster config")
		}
		if c.MaxRequestPerConn == 0 {
			c.MaxRequestPerConn = v2.DefaultMaxRequestPerConn
			log.StartLogger.Infof("[config] [parse cluster] max_request_per_conn is not specified, use default value %d",
				v2.DefaultMaxRequestPerConn)
		}
		if c.ConnBufferLimitBytes == 0 {
			c.ConnBufferLimitBytes = v2.DefaultConnBufferLimitBytes
			log.StartLogger.Infof("[config] [parse cluster] conn_buffer_limit_bytes is not specified, use default value %d",
				v2.DefaultConnBufferLimitBytes)
		}
		if c.LBSubSetConfig.FallBackPolicy > 2 {
			log.StartLogger.Errorf("[config] [parse cluster] lb subset config 's fall back policy set error. " +
				"For 0, represent NO_FALLBACK" +
				"For 1, represent ANY_ENDPOINT" +
				"For 2, represent DEFAULT_SUBSET")
			c.LBSubSetConfig.FallBackPolicy = 0
		}
		c.Hosts = parseHostConfig(c.Hosts)
		clusterV2Map[c.Name] = c.Hosts
		pClusters = append(pClusters, c)
	}
	// trigger all callbacks
	if cbs, ok := configParsedCBMaps[ParseCallbackKeyCluster]; ok {
		for _, cb := range cbs {
			cb(pClusters, false)
		}
	}

	return pClusters, clusterV2Map
}

func parseHostConfig(hosts []v2.Host) (hs []v2.Host) {
	for _, host := range hosts {
		host.Weight = transHostWeight(host.Weight)
		hs = append(hs, host)
	}
	return
}

func transHostWeight(weight uint32) uint32 {
	if weight > v2.MaxHostWeight {
		return v2.MaxHostWeight
	}
	if weight < v2.MinHostWeight {
		return v2.MinHostWeight
	}
	return weight
}

var logLevelMap = map[string]log.Level{
	"TRACE": log.TRACE,
	"DEBUG": log.DEBUG,
	"FATAL": log.FATAL,
	"ERROR": log.ERROR,
	"WARN":  log.WARN,
	"INFO":  log.INFO,
}

func ParseLogLevel(level string) log.Level {
	if logLevel, ok := logLevelMap[level]; ok {
		return logLevel
	}
	return log.INFO
}

func GetAddrIp(addr net.Addr) net.IP {
	switch addr.(type) {
	case *net.UDPAddr:
		return addr.(*net.UDPAddr).IP
	case *net.TCPAddr:
		return addr.(*net.TCPAddr).IP
	default:
		return nil
	}
}

func GetAddrPort(addr net.Addr) int {
	switch addr.(type) {
	case *net.UDPAddr:
		return addr.(*net.UDPAddr).Port
	case *net.TCPAddr:
		return addr.(*net.TCPAddr).Port
	default:
		return 0
	}
}

// ParseListenerConfig
func ParseListenerConfig(lc *v2.Listener, inheritListeners []net.Listener, inheritPacketConn []net.PacketConn) *v2.Listener {
	log.DefaultLogger.Infof("parsing listen config:%s", lc.Network)
	if lc.Network == "" {
		lc.Network = "tcp"
	}
	lc.Network = strings.ToLower(lc.Network)
	// Listener Config maybe not generated from json string
	if lc.Addr == nil {
		var addr net.Addr
		var err error
		switch lc.Network {
		case "udp":
			addr, err = net.ResolveUDPAddr("udp", lc.AddrConfig)
		case "unix":
			addr, err = net.ResolveUnixAddr("unix", lc.AddrConfig)
		case "tcp":
			addr, err = net.ResolveTCPAddr("tcp", lc.AddrConfig)
		default:
			err = fmt.Errorf("unknown listen type: %s , only support tcp,udp,unix", lc.Network)
		}
		if err != nil {
			log.StartLogger.Fatalf("[config] [parse listener] Address not valid: %v", lc.AddrConfig)
		}
		lc.Addr = addr
	}

	var old net.Listener
	var old_pc *net.PacketConn
	addr := lc.Addr
	// try inherit legacy listener or packet connection
	switch lc.Network {
	case "udp":
		for i, il := range inheritPacketConn {
			if il == nil {
				continue
			}
			tl := il.(*net.UDPConn)
			ilAddr, err := net.ResolveUDPAddr("udp", tl.LocalAddr().String())
			if err != nil {
				log.StartLogger.Fatalf("[config] [parse listener] inheritListener not valid: %s", tl.LocalAddr().String())
			}

			if GetAddrPort(addr) != ilAddr.Port {
				continue
			}

			ip := GetAddrIp(addr)
			if (ip.IsUnspecified() && ilAddr.IP.IsUnspecified()) ||
				(ip.IsLoopback() && ilAddr.IP.IsLoopback()) ||
				ip.Equal(ilAddr.IP) {
				log.StartLogger.Infof("[config] [parse listener] [udp] inherit packetConn addr: %s", lc.AddrConfig)
				old_pc = &il
				inheritPacketConn[i] = nil
				break
			}
		}

	case "unix":
		for i, il := range inheritListeners {
			if il == nil {
				continue
			}

			if _, ok := il.(*net.UnixListener); !ok {
				continue
			}
			unixls := il.(*net.UnixListener)

			path := unixls.Addr().String()
			if addr.String() == path {
				log.StartLogger.Infof("[config] [parse listener] [unix] inherit listener addr: %s", lc.AddrConfig)
				old = unixls
				inheritListeners[i] = nil
			}
		}
	case "tcp":
		// default tcp
		for i, il := range inheritListeners {
			if il == nil {
				continue
			}
			var tl *net.TCPListener
			var ok bool
			if tl, ok = il.(*net.TCPListener); !ok {
				continue
			}
			ilAddr, err := net.ResolveTCPAddr("tcp", tl.Addr().String())
			if err != nil {
				log.StartLogger.Fatalf("[config] [parse listener] inheritListener not valid: %s", tl.Addr().String())
			}

			if GetAddrPort(addr) != ilAddr.Port {
				continue
			}

			ip := GetAddrIp(addr)
			if (ip.IsUnspecified() && ilAddr.IP.IsUnspecified()) ||
				(ip.IsLoopback() && ilAddr.IP.IsLoopback()) ||
				ip.Equal(ilAddr.IP) {
				log.StartLogger.Infof("[config] [parse listener] [tcp] inherit listener addr: %s", lc.AddrConfig)
				old = tl
				inheritListeners[i] = nil
				break
			}
		}
	}

	lc.InheritListener = old
	lc.InheritPacketConn = old_pc
	return lc
}

func ParseRouterConfiguration(c *v2.FilterChain) (*v2.RouterConfiguration, error) {
	routerConfiguration := &v2.RouterConfiguration{}
	for _, f := range c.Filters {
		if f.Type == v2.CONNECTION_MANAGER {
			data, err := json.Marshal(f.Config)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, routerConfiguration); err != nil {
				return nil, err
			}
		}
	}
	return routerConfiguration, nil

}

// ParseServerConfig
func ParseServerConfig(c *v2.ServerConfig) *v2.ServerConfig {
	setMaxProcsWithProcessor(c.Processor)

	process := runtime.GOMAXPROCS(0)
	// trigger processor callbacks
	if cbs, ok := configParsedCBMaps[ParseCallbackKeyProcessor]; ok {
		for _, cb := range cbs {
			cb(process, true)
		}
	}
	return c
}

func setMaxProcsWithProcessor(procs interface{}) {
	// env variable has the highest priority.
	if n, _ := strconv.Atoi(os.Getenv("GOMAXPROCS")); n > 0 {
		runtime.GOMAXPROCS(n)
		return
	}
	if procs == nil {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return
	}

	intfunc := func(p int) {
		// use manual setting
		// no judge p > runtime.NumCPU(), because some situation maybe need this, such as multi io
		if p < 1 {
			p = runtime.NumCPU()
		}
		runtime.GOMAXPROCS(p)
	}

	strfunc := func(p string) {
		if strings.EqualFold(p, "auto") {
			// auto config with real cpu core or limit cpu core
			maxprocs.Set(maxprocs.Logger(log.DefaultLogger.Infof))
			return
		}

		pi, err := strconv.Atoi(p)
		if err != nil {
			log.DefaultLogger.Warnf("[configuration] server.processor is not stringnumber, use auto config.")
			maxprocs.Set(maxprocs.Logger(log.DefaultLogger.Infof))
			return
		}
		intfunc(pi)
	}

	switch processor := procs.(type) {
	case string:
		strfunc(processor)
	case int:
		intfunc(processor)
	case float64:
		intfunc(int(processor))
	default:
		log.StartLogger.Fatalf("unsupport serverconfig processor type %v, must be int or string.", reflect.TypeOf(procs))
	}
}

var listenerFilterFactoryMap = sync.Map{}

// AddOrUpdateListenerFilterFactories adds or updates the listener filter factories of a listener
func AddOrUpdateListenerFilterFactories(listenerName string, configs []v2.Filter) []api.ListenerFilterChainFactory {
	if listenerName == "" || len(configs) == 0 {
		return nil
	}

	var factories []api.ListenerFilterChainFactory

	for _, c := range configs {
		sfcc, err := api.CreateListenerFilterChainFactory(c.Type, c.Config)
		if err != nil {
			log.DefaultLogger.Errorf("[config] AddOrUpdateListenerFilterFactories failed, type: %s, error: %v", c.Type, err)
			continue
		}
		if sfcc != nil {
			factories = append(factories, sfcc)
		}
	}

	if len(factories) == 0 {
		log.DefaultLogger.Errorf("[config] AddOrUpdateListenerFilterFactories factories len is 0, listenerName: %v", listenerName)
		return nil
	}

	listenerFilterFactoryMap.Store(listenerName, factories)
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[config] AddOrUpdateListenerFilterFactories store listener factory, name: %v, config: %v",
			listenerName, configs)
	}

	return factories
}

// GetListenerFilterFactories returns a listener filter factory by filter.Type
func GetListenerFilterFactories(listenerName string) []api.ListenerFilterChainFactory {
	if listenerName == "" {
		return nil
	}

	if v, ok := listenerFilterFactoryMap.Load(listenerName); ok {
		return v.([]api.ListenerFilterChainFactory)
	}

	return nil
}

var networkFilterFactoryMap = sync.Map{}

// AddOrUpdateNetworkFilterFactories adds or updates the network filter factories of a listener
func AddOrUpdateNetworkFilterFactories(listenerName string, ln *v2.Listener) []api.NetworkFilterChainFactory {
	if ln == nil || listenerName == "" {
		log.DefaultLogger.Errorf("[config] network filter create failed, error: nil listener or empty name")
		return nil
	}

	var factories []api.NetworkFilterChainFactory
	c := ln.FilterChains[0]
	for _, f := range c.Filters {
		factory, err := api.CreateNetworkFilterChainFactory(f.Type, f.Config)
		if err != nil {
			log.DefaultLogger.Errorf("[config] network filter create failed, type:%s, error: %v", f.Type, err)
			continue
		}
		if initializer, ok := factory.(api.FactoryInitializer); ok {
			if err := initializer.Init(ln); err != nil {
				log.DefaultLogger.Errorf("[config] network filter init failed, type:%s, error:%v", f.Type, err)
				continue
			}
		}
		if factory != nil {
			factories = append(factories, factory)
		}
	}

	if len(factories) == 0 {
		log.DefaultLogger.Errorf("[config] network filter factories len is 0, listener: %+v", ln)
		return nil
	}

	networkFilterFactoryMap.Store(listenerName, factories)
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[config] AddOrUpdateNetworkFilterFactories store network filter factories, name: %v", listenerName)
	}

	return factories
}

// GetNetworkFilterFactories returns a network filter factory by filter.Type
func GetNetworkFilterFactories(listenerName string) []api.NetworkFilterChainFactory {
	if listenerName == "" {
		return nil
	}

	if v, ok := networkFilterFactoryMap.Load(listenerName); ok {
		return v.([]api.NetworkFilterChainFactory)
	}

	return nil
}
