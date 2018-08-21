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

package mosn

import (
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	"github.com/alipay/sofa-mosn/pkg/xds"
)

// Mosn class which wrapper server
type Mosn struct {
	servers []server.Server
}

// NewMosn
// Create server from mosn config
func NewMosn(c *config.MOSNConfig) *Mosn {
	m := &Mosn{}
	mode := c.Mode()

	if mode == config.Xds {
		servers := make([]config.ServerConfig, 0, 1)
		server := config.ServerConfig{
			DefaultLogPath:  "stdout",
			DefaultLogLevel: "INFO",
		}
		servers = append(servers, server)
		c.Servers = servers
	} else {
		if c.ClusterManager.Clusters == nil || len(c.ClusterManager.Clusters) == 0 {
			if !c.ClusterManager.AutoDiscovery {
				log.StartLogger.Fatalln("no cluster found and cluster manager doesn't support auto discovery")
			}
		}
	}

	srvNum := len(c.Servers)

	if srvNum == 0 {
		log.StartLogger.Fatalln("no server found")
	} else if srvNum > 1 {
		log.StartLogger.Fatalln("multiple server not supported yet, got ", srvNum)
	}
	//get inherit fds
	inheritListeners := getInheritListeners()

	//cluster manager filter
	cmf := &clusterManagerFilter{}

	// parse cluster all in one
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)

	for _, serverConfig := range c.Servers {
		//1. server config prepare
		//server config
		sc := config.ParseServerConfig(&serverConfig)

		// init default log
		server.InitDefaultLogger(sc)

		var srv server.Server
		if mode == config.Xds {
			cmf := &clusterManagerFilter{}
			cm := cluster.NewClusterManager(nil, nil, nil, true, false)
			srv = server.NewServer(sc, cmf, cm)

		} else {
			//create cluster manager
			cm := cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery, c.ClusterManager.RegistryUseHealthCheck)
			//initialize server instance
			srv = server.NewServer(sc, cmf, cm)

			//add listener
			if serverConfig.Listeners == nil || len(serverConfig.Listeners) == 0 {
				log.StartLogger.Fatalln("no listener found")
			}

			for _, listenerConfig := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := config.ParseListenerConfig(&listenerConfig, inheritListeners)

				nfcf := getNetworkFilters(&lc.FilterChains[0])

				// Note: as we use fasthttp and net/http2.0, the IO we created in mosn should be disabled
				// in the future, if we realize these two protocol by-self, this this hack method should be removed
				lc.DisableConnIo = listenerDisableIO(&lc.FilterChains[0])

				// network filters
				if lc.HandOffRestoredDestinationConnections {
					srv.AddListener(lc, nil, nil)
					continue
				}

				//stream filters
				sfcf := getStreamFilters(listenerConfig.StreamFilters)
				config.SetGlobalStreamFilter(sfcf)
				srv.AddListener(lc, nfcf, sfcf)
			}
		}
		m.servers = append(m.servers, srv)
	}
	//parse service registry info
	config.ParseServiceRegistry(c.ServiceRegistry)

	//close legacy listeners
	for _, ln := range inheritListeners {
		if !ln.Remain {
			log.StartLogger.Println("close useless legacy listener:", ln.Addr)
			ln.InheritListener.Close()
		}
	}

	// set TransferTimeout
	network.TransferTimeout = server.GracefulTimeout
	// transfer old mosn connections
	go network.TransferServer(m.servers[0].Handler())

	return m
}

// Start mosn's server
func (m *Mosn) Start() {
	for _, srv := range m.servers {
		go srv.Start()
	}
}

// Close mosn's server
func (m *Mosn) Close() {
	for _, srv := range m.servers {
		srv.Close()
	}
}

// Start mosn project
// stap1. NewMosn
// step2. Start Mosn
func Start(c *config.MOSNConfig, serviceCluster string, serviceNode string) {
	log.StartLogger.Infof("start by config : %+v", c)

	wg := sync.WaitGroup{}
	wg.Add(1)

	Mosn := NewMosn(c)
	Mosn.Start()
	////get xds config
	xdsClient := xds.Client{}
	xdsClient.Start(c, serviceCluster, serviceNode)
	//
	////todo: daemon running
	wg.Wait()
	xdsClient.Stop()
}

// getNetworkFilter
// Used to parse proxy from config
func getNetworkFilters(c *v2.FilterChain) []types.NetworkFilterChainFactory {
	var factories []types.NetworkFilterChainFactory
	for _, f := range c.Filters {
		factory, err := filter.CreateNetworkFilterChainFactory(f.Name, f.Config)
		if err != nil {
			log.StartLogger.Fatalln("network filter create failed :", err)
		}
		factories = append(factories, factory)
	}
	return factories
}
func listenerDisableIO(c *v2.FilterChain) bool {
	for _, f := range c.Filters {
		if f.Name == v2.DEFAULT_NETWORK_FILTER {
			if downstream, ok := f.Config["downstream_protocol"]; ok {
				if downstream == string(protocol.HTTP2) || downstream == string(protocol.HTTP1) {
					return true
				}
			}
		}
	}
	return false
}

func getStreamFilters(configs []config.FilterConfig) []types.StreamFilterChainFactory {
	var factories []types.StreamFilterChainFactory

	for _, c := range configs {
		factory, err := filter.CreateStreamFilterChainFactory(c.Type, c.Config)
		if err != nil {
			log.StartLogger.Fatalln("stream filter create failed :", err)
		}
		factories = append(factories, factory)
	}
	return factories
}

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func getInheritListeners() []*v2.ListenerConfig {
	if os.Getenv("_MOSN_GRACEFUL_RESTART") == "true" {
		count, _ := strconv.Atoi(os.Getenv("_MOSN_INHERIT_FD"))
		listeners := make([]*v2.ListenerConfig, count)

		log.StartLogger.Infof("received %d inherit fds", count)

		for idx := 0; idx < count; idx++ {
			//because passed listeners fd's index starts from 3
			fd := uintptr(3 + idx)
			file := os.NewFile(fd, "")
			fileListener, err := net.FileListener(file)
			if err != nil {
				log.StartLogger.Errorf("recover listener from fd %d failed: %s", fd, err)
				continue
			}
			if listener, ok := fileListener.(*net.TCPListener); ok {
				listeners[idx] = &v2.ListenerConfig{Addr: listener.Addr(), InheritListener: listener}
			} else {
				log.StartLogger.Errorf("listener recovered from fd %d is not a tcp listener", fd)
			}
		}
		return listeners
	}
	return nil
}
