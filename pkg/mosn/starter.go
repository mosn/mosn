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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/server/keeper"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/mosn/pkg/xds"
	"mosn.io/pkg/utils"
)

// Mosn class which wrapper server
type Mosn struct {
	servers        []server.Server
	clustermanager types.ClusterManager
	routerManager  types.RouterManager
	config         *v2.MOSNConfig
	adminServer    admin.Server
	xdsClient      *xds.Client
	wg             sync.WaitGroup
	// for smooth upgrade. reconfigure
	inheritListeners  []net.Listener
	inheritPacketConn []net.PacketConn
	listenSockConn    net.Conn
}

// NewMosn
// Create server from mosn config
func NewMosn(c *v2.MOSNConfig) *Mosn {
	initializeDefaultPath(configmanager.GetConfigPath())
	initializePidFile(c.Pid)
	initializeTracing(c.Tracing)
	initializePlugin(c.Plugin.LogBase)

	store.SetMosnConfig(c)

	//get inherit fds
	inheritListeners, inheritPacketConn, listenSockConn, err := server.GetInheritListeners()
	if err != nil {
		log.StartLogger.Fatalf("[mosn] [NewMosn] getInheritListeners failed, exit")
	}
	if listenSockConn != nil {
		log.StartLogger.Infof("[mosn] [NewMosn] active reconfiguring")
		// set Mosn Active_Reconfiguring
		store.SetMosnState(store.Active_Reconfiguring)
		// parse MOSNConfig again
		c = configmanager.Load(configmanager.GetConfigPath())
	} else {
		log.StartLogger.Infof("[mosn] [NewMosn] new mosn created")
		// start init services
		if err := store.StartService(nil); err != nil {
			log.StartLogger.Fatalf("[mosn] [NewMosn] start service failed: %v,  exit", err)
		}
	}

	initializeMetrics(c.Metrics)

	m := &Mosn{
		config:            c,
		wg:                sync.WaitGroup{},
		inheritListeners:  inheritListeners,
		inheritPacketConn: inheritPacketConn,
		listenSockConn:    listenSockConn,
	}
	mode := c.Mode()

	if mode == v2.Xds {
		c.Servers = []v2.ServerConfig{
			{
				DefaultLogPath:  "stdout",
				DefaultLogLevel: "INFO",
			},
		}
	} else {
		if len(c.ClusterManager.Clusters) == 0 && !c.ClusterManager.AutoDiscovery {
			log.StartLogger.Fatalf("[mosn] [NewMosn] no cluster found and cluster manager doesn't support auto discovery")
		}
	}

	srvNum := len(c.Servers)

	if srvNum == 0 {
		log.StartLogger.Fatalf("[mosn] [NewMosn] no server found")
	} else if srvNum > 1 {
		log.StartLogger.Fatalf("[mosn] [NewMosn] multiple server not supported yet, got %d", srvNum)
	}

	//cluster manager filter
	cmf := &clusterManagerFilter{}

	// parse cluster all in one
	clusters, clusterMap := configmanager.ParseClusterConfig(c.ClusterManager.Clusters)
	// create cluster manager
	if mode == v2.Xds {
		m.clustermanager = cluster.NewClusterManagerSingleton(nil, nil)
	} else {
		m.clustermanager = cluster.NewClusterManagerSingleton(clusters, clusterMap)
	}

	// initialize the routerManager
	m.routerManager = router.NewRouterManager()

	for _, serverConfig := range c.Servers {
		//1. server config prepare
		//server config
		c := configmanager.ParseServerConfig(&serverConfig)

		// new server config
		sc := server.NewConfig(c)

		// init default log
		server.InitDefaultLogger(sc)

		var srv server.Server
		if mode == v2.Xds {
			srv = server.NewServer(sc, cmf, m.clustermanager)
		} else {
			//initialize server instance
			srv = server.NewServer(sc, cmf, m.clustermanager)

			//add listener
			if len(serverConfig.Listeners) == 0 {
				log.StartLogger.Fatalf("[mosn] [NewMosn] no listener found")
			}

			for idx, _ := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := configmanager.ParseListenerConfig(&serverConfig.Listeners[idx], inheritListeners, inheritPacketConn)
				// deprecated: keep compatible for route config in listener's connection_manager
				deprecatedRouter, err := configmanager.ParseRouterConfiguration(&lc.FilterChains[0])
				if err != nil {
					log.StartLogger.Fatalf("[mosn] [NewMosn] compatible router: %v", err)
				}
				if deprecatedRouter.RouterConfigName != "" {
					m.routerManager.AddOrUpdateRouters(deprecatedRouter)
				}
				if _, err := srv.AddListener(lc); err != nil {
					log.StartLogger.Fatalf("[mosn] [NewMosn] AddListener error:%s", err.Error())
				}
			}
			// Add Router Config
			for _, routerConfig := range serverConfig.Routers {
				if routerConfig.RouterConfigName != "" {
					m.routerManager.AddOrUpdateRouters(routerConfig)
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
		if _, err := m.listenSockConn.Write([]byte{0}); err != nil {
			log.StartLogger.Fatalf("[mosn] [NewMosn] graceful failed, exit")
		}

		m.listenSockConn.Close()

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
		configmanager.DumpConfigHandler()
	}, nil)

	// start reconfig domain socket
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
	// start mosn feature
	featuregate.StartInit()
	// TODO: remove it
	//parse service registry info
	log.StartLogger.Infof("mosn parse registry info")
	configmanager.ParseServiceRegistry(m.config.ServiceRegistry)

	log.StartLogger.Infof("mosn parse extend config")
	configmanager.ParseConfigExtend(m.config.Extend)

	// beforestart starts transfer connection and non-proxy listeners
	log.StartLogger.Infof("mosn prepare for start")
	m.beforeStart()

	// start mosn server
	log.StartLogger.Infof("mosn start server")
	for _, srv := range m.servers {

		// TODO
		// This can't be deleted, otherwise the code behind is equivalent at
		// utils.GoWithRecover(func() {
		//	 m.servers[0].Start()
		// },
		srv := srv

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
func Start(c *v2.MOSNConfig) {
	//log.StartLogger.Infof("[mosn] [start] start by config : %+v", c)
	Mosn := NewMosn(c)
	Mosn.Start()
	Mosn.wg.Wait()
}

func initializeTracing(config v2.TracingConfig) {
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

func initializeMetrics(config v2.MetricsConfig) {
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

func initializePlugin(log string) {
	if log == "" {
		log = types.MosnLogBasePath
	}
	plugin.InitPlugin(log)
}

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}
