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
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/connectionmanager"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	"github.com/alipay/sofa-mosn/pkg/xds"
)

// Mosn class which wrapper server
type Mosn struct {
	servers        []server.Server
	clustermanager types.ClusterManager
	routerManager  types.RouterManager
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
	// create cluster manager
	if mode == config.Xds {
		m.clustermanager = cluster.NewClusterManager(nil, nil, nil, true, false)
	} else {
		m.clustermanager = cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery, c.ClusterManager.RegistryUseHealthCheck)
	}

	// initialize the routerManager
	m.routerManager = router.NewRouterManager()

	for _, serverConfig := range c.Servers {
		//1. server config prepare
		//server config
		sc := config.ParseServerConfig(&serverConfig)

		// init default log
		server.InitDefaultLogger(sc)

		var srv server.Server
		if mode == config.Xds {
			srv = server.NewServer(sc, cmf, m.clustermanager)

		} else {
			//initialize server instance
			srv = server.NewServer(sc, cmf, m.clustermanager)

			//add listener
			if serverConfig.Listeners == nil || len(serverConfig.Listeners) == 0 {
				log.StartLogger.Fatalln("no listener found")
			}

			for _, listenerConfig := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := config.ParseListenerConfig(&listenerConfig, inheritListeners)
				lc.DisableConnIo = config.GetListenerDisableIO(&lc.FilterChains[0])

				// parse routers from connection_manager filter and add it the routerManager
				if routerConfig := config.ParseRouterConfiguration(&lc.FilterChains[0]); routerConfig.RouterConfigName != "" {
					m.routerManager.AddOrUpdateRouters(routerConfig)
				}

				var nfcf []types.NetworkFilterChainFactory
				var sfcf []types.StreamFilterChainFactory

				// Note: as we use fasthttp and net/http2.0, the IO we created in mosn should be disabled
				// network filters
				if !lc.HandOffRestoredDestinationConnections {
					// network and stream filters
					nfcf = config.GetNetworkFilters(&lc.FilterChains[0])
					sfcf = config.GetStreamFilters(lc.StreamFilters)
				}

				_, err := srv.AddListener(lc, nfcf, sfcf)
				if err != nil {
					log.StartLogger.Fatalf("AddListener error:%s", err.Error())
				}
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
	// transfer old mosn mertrics, none-block
	go stats.TransferServer(server.GracefulTimeout, nil)

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
	m.clustermanager.Destory()
}

// Start mosn project
// step1. NewMosn
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

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func getInheritListeners() []*v2.Listener {
	if os.Getenv(types.GracefulRestart) == "true" {
		count, _ := strconv.Atoi(os.Getenv(types.InheritFd))
		listeners := make([]*v2.Listener, count)

		log.StartLogger.Infof("received %d inherit fds", count)

		for idx := 0; idx < count; idx++ {
			func() {
				//because passed listeners fd's index starts from 3
				fd := uintptr(3 + idx)
				file := os.NewFile(fd, "")
				if file == nil {
					log.StartLogger.Errorf("create new file from fd %d failed", fd)
					return
				}
				defer file.Close()

				fileListener, err := net.FileListener(file)
				if err != nil {
					log.StartLogger.Errorf("recover listener from fd %d failed: %s", fd, err)
					return
				}
				if listener, ok := fileListener.(*net.TCPListener); ok {
					listeners[idx] = &v2.Listener{Addr: listener.Addr(), InheritListener: listener}
				} else {
					log.StartLogger.Errorf("listener recovered from fd %d is not a tcp listener", fd)
				}
			}()
		}
		return listeners
	}
	return nil
}
