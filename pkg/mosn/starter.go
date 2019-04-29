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
	"sync"

	admin "github.com/alipay/sofa-mosn/pkg/admin/server"
	"github.com/alipay/sofa-mosn/pkg/admin/store"
	v2 "github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/connectionmanager"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/alipay/sofa-mosn/pkg/metrics/shm"
	"github.com/alipay/sofa-mosn/pkg/metrics/sink"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/server/keeper"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	"github.com/alipay/sofa-mosn/pkg/xds"
)

// Mosn class which wrapper server
type Mosn struct {
	servers        []server.Server
	clustermanager types.ClusterManager
	routerManager  types.RouterManager
	config         *config.MOSNConfig
	adminServer    admin.Server
}

// NewMosn
// Create server from mosn config
func NewMosn(c *config.MOSNConfig) *Mosn {
	initializeDefaultPath(config.GetConfigPath())
	initializePidFile(c.Pid)
	initializeTracing(c.Tracing)
	initializeMetrics(c.Metrics)

	//get inherit fds
	inheritListeners, reconfigure, err := server.GetInheritListeners()
	if err != nil {
		log.StartLogger.Fatalln("getInheritListeners failed, exit")
	}
	if reconfigure != nil {
		// set Mosn Active_Reconfiguring
		store.SetMosnState(store.Active_Reconfiguring)
		// parse MOSNConfig again
		c = config.Load(config.GetConfigPath())
	} else {
		// start init services
		if err := store.StartService(nil); err != nil {
			log.StartLogger.Fatalln("start service failed: %v,  exit", err)
		}
	}

	m := &Mosn{
		config: c,
	}
	mode := c.Mode()

	if mode == config.Xds {
		servers := make([]v2.ServerConfig, 0, 1)
		server := v2.ServerConfig{
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
		c := config.ParseServerConfig(&serverConfig)

		// new server config
		sc := server.NewConfig(c)

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

			for idx, _ := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := config.ParseListenerConfig(&serverConfig.Listeners[idx], inheritListeners)
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
					log.StartLogger.Errorf("AddListener error:%s", err.Error())
				}
			}
		}
		m.servers = append(m.servers, srv)
	}

	//parse service registry info
	config.ParseServiceRegistry(c.ServiceRegistry)

	// start adminApi
	m.adminServer = admin.Server{}
	m.adminServer.Start(m.config)

	// SetTransferTimeout
	network.SetTransferTimeout(server.GracefulTimeout)

	if store.GetMosnState() == store.Active_Reconfiguring {
		// start other services
		if err := store.StartService(inheritListeners); err != nil {
			log.StartLogger.Fatalln("start service failed: %v,  exit", err)
		}

		// notify old mosn to transfer connection
		if _, err := reconfigure.Write([]byte{0}); err != nil {
			log.StartLogger.Fatalln("graceful failed, exit")
		}

		reconfigure.Close()

		// transfer old mosn connections
		go network.TransferServer(m.servers[0].Handler())
	} else {
		// start other services
		if err := store.StartService(nil); err != nil {
			log.StartLogger.Fatalln("start service failed: %v,  exit", err)
		}
		store.SetMosnState(store.Running)
	}

	//close legacy listeners
	for _, ln := range inheritListeners {
		if ln != nil {
			log.StartLogger.Printf("close useless legacy listener: %s", ln.Addr().String())
			ln.Close()
		}
	}

	// start dump config process
	go config.DumpConfigHandler()

	// start reconfigure domain socket
	go server.ReconfigureHandler()

	return m
}

// Start mosn's server
func (m *Mosn) Start() {
	// start mosn server
	for _, srv := range m.servers {
		go srv.Start()
	}
}

// Close mosn's server
func (m *Mosn) Close() {
	// close service
	store.StopService()

	// stop reconfigure domain socket
	server.StopReconfigureHandler()

	// stop mosn server
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

func initializeTracing(config config.TracingConfig) {
	if config.Enable && config.Tracer != "" {
		tracer := trace.CreateTracer(config.Tracer)
		if tracer != nil {
			trace.SetTracer(tracer)
		} else {
			log.DefaultLogger.Errorf("Unable to recognise tracing implementation %s, tracing functionality is turned off.", config.Tracer)
			trace.DisableTracing()
			return
		}
		trace.EnableTracing()
	} else {
		trace.DisableTracing()
	}
}

func initializeMetrics(config config.MetricsConfig) {
	// init shm zone
	if config.ShmZone != "" && config.ShmSize > 0 {
		shm.InitDefaultMetricsZone(config.ShmZone, int(config.ShmSize))
	}

	// set metrics package
	statsMatcher := config.StatsMatcher
	metrics.SetStatsMatcher(statsMatcher.RejectAll, statsMatcher.ExclusionLabels, statsMatcher.ExclusionKeys)
	// create sinks
	for _, cfg := range config.SinkConfigs {
		_, err := sink.CreateMetricsSink(cfg.Type, cfg.Config)
		// abort
		if err != nil {
			log.StartLogger.Errorf("%s. %v metrics sink is turned off", err, cfg.Type)
			return
		}
	}
}

func initializePidFile(pid string) {
	keeper.SetPid(pid)
}

func initializeDefaultPath(path string) {
	types.InitDefaultPath(path)
}

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}
