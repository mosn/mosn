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
	"sync"

	admin "mosn.io/mosn/pkg/admin/server"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/api/v2"
	"mosn.io/mosn/pkg/config"
	_ "mosn.io/mosn/pkg/filter/network/connectionmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/server/keeper"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/mosn/pkg/utils"
	"mosn.io/mosn/pkg/xds"
)

// Mosn class which wrapper server
type Mosn struct {
	servers        []server.Server
	clustermanager types.ClusterManager
	routerManager  types.RouterManager
	config         *config.MOSNConfig
	adminServer    admin.Server
	xdsClient      *xds.Client
	wg             sync.WaitGroup
	// for smooth upgrade. reconfigure
	inheritListeners []net.Listener
	reconfigure      net.Conn
}

// NewMosn
// Create server from mosn config
func NewMosn(c *config.MOSNConfig) *Mosn {
	initializeDefaultPath(config.GetConfigPath())
	initializePidFile(c.Pid)
	initializeTracing(c.Tracing)

	//get inherit fds
	inheritListeners, reconfigure, err := server.GetInheritListeners()
	if err != nil {
		log.StartLogger.Fatalln("[mosn] [NewMosn] getInheritListeners failed, exit")
	}
	if reconfigure != nil {
		log.StartLogger.Infof("[mosn] [NewMosn] active reconfiguring")
		// set Mosn Active_Reconfiguring
		store.SetMosnState(store.Active_Reconfiguring)
		// parse MOSNConfig again
		c = config.Load(config.GetConfigPath())
	} else {
		log.StartLogger.Infof("[mosn] [NewMosn] new mosn created")
		// start init services
		if err := store.StartService(nil); err != nil {
			log.StartLogger.Fatalf("[mosn] [NewMosn] start service failed: %v,  exit", err)
		}
	}

	initializeMetrics(c.Metrics)

	m := &Mosn{
		config:           c,
		wg:               sync.WaitGroup{},
		inheritListeners: inheritListeners,
		reconfigure:      reconfigure,
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
				log.StartLogger.Fatalln("[mosn] [NewMosn] no cluster found and cluster manager doesn't support auto discovery")
			}

		}
	}

	srvNum := len(c.Servers)

	if srvNum == 0 {
		log.StartLogger.Fatalln("[mosn] [NewMosn] no server found")
	} else if srvNum > 1 {
		log.StartLogger.Fatalln("[mosn] [NewMosn] multiple server not supported yet, got ", srvNum)
	}

	//cluster manager filter
	cmf := &clusterManagerFilter{}

	// parse cluster all in one
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	// create cluster manager
	if mode == config.Xds {
		m.clustermanager = cluster.NewClusterManagerSingleton(nil, nil)
	} else {
		m.clustermanager = cluster.NewClusterManagerSingleton(clusters, clusterMap)
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
				log.StartLogger.Fatalln("[mosn] [NewMosn] no listener found")
			}

			for idx, _ := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := config.ParseListenerConfig(&serverConfig.Listeners[idx], inheritListeners)

				// parse routers from connection_manager filter and add it the routerManager
				if routerConfig := config.ParseRouterConfiguration(&lc.FilterChains[0]); routerConfig.RouterConfigName != "" {
					m.routerManager.AddOrUpdateRouters(routerConfig)
				}

				var nfcf []types.NetworkFilterChainFactory
				var sfcf []types.StreamFilterChainFactory

				// Note: as we use fasthttp and net/http2.0, the IO we created in mosn should be disabled
				// network filters
				if !lc.UseOriginalDst {
					// network and stream filters
					nfcf = config.GetNetworkFilters(&lc.FilterChains[0])
					sfcf = config.GetStreamFilters(lc.StreamFilters)
				}

				_, err := srv.AddListener(lc, nfcf, sfcf)
				if err != nil {
					log.StartLogger.Fatalf("[mosn] [NewMosn] AddListener error:%s", err.Error())
				}
			}
		}
		m.servers = append(m.servers, srv)
	}

	return m
}

// beforeStart prepares some actions before mosn start proxy listener
func (m *Mosn) beforeStart() {
	// start adminApi
	m.adminServer = admin.Server{}
	m.adminServer.Start(m.config)

	// SetTransferTimeout
	network.SetTransferTimeout(server.GracefulTimeout)

	if store.GetMosnState() == store.Active_Reconfiguring {
		// start other services
		if err := store.StartService(m.inheritListeners); err != nil {
			log.StartLogger.Fatalf("[mosn] [NewMosn] start service failed: %v,  exit", err)
		}

		// notify old mosn to transfer connection
		if _, err := m.reconfigure.Write([]byte{0}); err != nil {
			log.StartLogger.Fatalln("[mosn] [NewMosn] graceful failed, exit")
		}

		m.reconfigure.Close()

		// transfer old mosn connections
		utils.GoWithRecover(func() {
			network.TransferServer(m.servers[0].Handler())
		}, nil)
	} else {
		// start other services
		if err := store.StartService(nil); err != nil {
			log.StartLogger.Fatalf("[mosn] [NewMosn] start service failed: %v,  exit", err)
		}
		store.SetMosnState(store.Running)
	}

	//close legacy listeners
	for _, ln := range m.inheritListeners {
		if ln != nil {
			log.StartLogger.Infof("[mosn] [NewMosn] close useless legacy listener: %s", ln.Addr().String())
			ln.Close()
		}
	}

	// start dump config process
	utils.GoWithRecover(func() {
		config.DumpConfigHandler()
	}, nil)

	// start reconfigure domain socket
	utils.GoWithRecover(func() {
		server.ReconfigureHandler()
	}, nil)
}

// Start mosn's server
func (m *Mosn) Start() {
	m.wg.Add(1)
	// Start XDS if configured
	log.StartLogger.Infof("mosn start xds client")
	m.xdsClient = &xds.Client{}
	utils.GoWithRecover(func() {
		m.xdsClient.Start(m.config)
	}, nil)
	// TODO: remove it
	//parse service registry info
	log.StartLogger.Infof("mosn parse registry info")
	config.ParseServiceRegistry(m.config.ServiceRegistry)

	// beforestart starts transfer connection and non-proxy listeners
	log.StartLogger.Infof("mosn prepare for start")
	m.beforeStart()

	// start mosn server
	log.StartLogger.Infof("mosn start server")
	for _, srv := range m.servers {
		utils.GoWithRecover(func() {
			srv.Start()
		}, nil)
	}
}

// Close mosn's server
func (m *Mosn) Close() {
	// close service
	store.CloseService()

	// stop reconfigure domain socket
	server.StopReconfigureHandler()

	// stop mosn server
	for _, srv := range m.servers {
		srv.Close()
	}
	m.xdsClient.Stop()
	m.clustermanager.Destroy()
	m.wg.Done()
}

// Start mosn project
// step1. NewMosn
// step2. Start Mosn
func Start(c *config.MOSNConfig) {
	log.StartLogger.Infof("[mosn] [start] start by config : %+v", c)
	Mosn := NewMosn(c)
	Mosn.Start()
	Mosn.wg.Wait()
}

func initializeTracing(config config.TracingConfig) {
	if config.Enable && config.Driver != "" {
		err := trace.Init(config.Driver, config.Config)
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init tracing] init driver '%s' failed: %s, tracing functionality is turned off.", config.Driver, err)
			trace.Disable()
			return
		}
		log.StartLogger.Infof("[mosn] [init tracing] enable tracing")
		trace.Enable()
	} else {
		log.StartLogger.Infof("[mosn] [init tracing] disbale tracing")
		trace.Disable()
	}
}

func initializeMetrics(config config.MetricsConfig) {
	// init shm zone
	if config.ShmZone != "" && config.ShmSize > 0 {
		shm.InitDefaultMetricsZone(config.ShmZone, int(config.ShmSize), store.GetMosnState() != store.Active_Reconfiguring)
	}

	// set metrics package
	statsMatcher := config.StatsMatcher
	metrics.SetStatsMatcher(statsMatcher.RejectAll, statsMatcher.ExclusionLabels, statsMatcher.ExclusionKeys)
	// create sinks
	for _, cfg := range config.SinkConfigs {
		_, err := sink.CreateMetricsSink(cfg.Type, cfg.Config)
		// abort
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init metrics] %s. %v metrics sink is turned off", err, cfg.Type)
			return
		}
		log.StartLogger.Infof("[mosn] [init metrics] create metrics sink: %v", cfg.Type)
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
