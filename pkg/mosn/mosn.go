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
	"encoding/json"
	"errors"
	"net"
	"time"

	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/ewma"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/server/pid"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/utils"
)

// UpgradeData stores data that are used to smooth upgrade
type UpgradeData struct {
	InheritListeners  []net.Listener
	InheritPacketConn []net.PacketConn
	ListenSockConn    net.Conn
}

type Mosn struct {
	isFromUpgrade  bool // hot upgrade from old MOSN
	Upgrade        UpgradeData
	Clustermanager types.ClusterManager
	RouterManager  types.RouterManager
	Config         *v2.MOSNConfig
	// internal data
	servers   []server.Server
	xdsClient *istio.ADSClient
}

// create an empty mosn
func NewMosn() *Mosn {
	log.StartLogger.Infof("[mosn start] create an empty mosn structure")
	m := &Mosn{
		Upgrade: UpgradeData{},
	}
	return m
}

// whether Mosn is hot upgraded from an old MOSN
func (m *Mosn) IsFromUpgrade() bool {
	return m.isFromUpgrade
}

// generate mosn structure members
func (m *Mosn) Init(c *v2.MOSNConfig) error {
	if c.CloseGraceful {
		c.DisableUpgrade = true
	}
	if err := m.inheritConfig(c); err != nil {
		return err
	}

	log.StartLogger.Infof("[mosn start] init the members of the mosn")

	// after inherit config,
	// since metrics need the isFromUpgrade flag in Mosn
	m.initializeMetrics()
	m.initClusterManager()
	m.initServer()

	// set the mosn config finally
	configmanager.SetMosnConfig(m.Config)
	return nil
}

// receive from old mosn
// stage manager will stop the new mosn when returning error
func (m *Mosn) inheritHandler() error {
	var err error
	m.Upgrade.InheritListeners, m.Upgrade.InheritPacketConn, m.Upgrade.ListenSockConn, err = server.GetInheritListeners()
	if err != nil {
		log.StartLogger.Errorf("[mosn] [NewMosn] getInheritListeners failed, exiting, err:%v", err)
		return err
	}
	log.StartLogger.Infof("[mosn] [NewMosn] active reconfiguring")
	// parse MOSNConfig again
	c := configmanager.Load(configmanager.GetConfigPath())
	if c.InheritOldMosnconfig {
		err = inheritFunc(c)
		if err != nil {
			m.Upgrade.ListenSockConn.Close()
			log.StartLogger.Errorf("[mosn] [NewMosn] InheritConfig failed, exiting, err: %v", err)
			return err
		}
	}
	if c.CloseGraceful {
		c.DisableUpgrade = true
	}
	m.Config = c
	return nil
}

// replace your own inherit func with default inherit func
func InitInheritFunc(f func(c *v2.MOSNConfig) error) {
	inheritFunc = f
}

var inheritFunc = func(c *v2.MOSNConfig) error {
	// inherit old mosn config
	configData, err := server.GetInheritConfig()
	if err != nil {
		return nil
	}

	oldMosnConfig := &v2.MOSNConfig{}
	err = json.Unmarshal(configData, oldMosnConfig)
	if err != nil {
		return err
	}

	log.StartLogger.Debugf("[mosn] [NewMosn] old mosn config: %v", oldMosnConfig)
	c.Servers = oldMosnConfig.Servers
	c.ClusterManager = oldMosnConfig.ClusterManager
	c.Extends = oldMosnConfig.Extends

	return nil
}

// inherit listener fds / config from old mosn when it exists,
// use the local config by default,
// stop the new mosn when error happens
func (m *Mosn) inheritConfig(c *v2.MOSNConfig) (err error) {
	m.Config = c
	server.EnableInheritOldMosnconfig(c.InheritOldMosnconfig)

	// default is graceful mode, turn graceful off by set it to false
	if !c.DisableUpgrade && server.IsReconfigure() {
		m.isFromUpgrade = true
		if err = m.inheritHandler(); err != nil {
			return
		}
	}
	log.StartLogger.Infof("[mosn] [NewMosn] new mosn created")
	// start init services
	if err = store.StartService(m.Upgrade.InheritListeners); err != nil {
		log.StartLogger.Errorf("[mosn] [NewMosn] start service failed: %v, exit", err)
	}
	return
}

func (m *Mosn) initializeMetrics() {
	metrics.FlushMosnMetrics = true
	config := m.Config.Metrics

	// init shm zone
	if config.ShmZone != "" && config.ShmSize > 0 {
		shm.InitDefaultMetricsZone(config.ShmZone, int(config.ShmSize), !m.IsFromUpgrade())
	}

	// set metrics package
	statsMatcher := config.StatsMatcher
	metrics.SetStatsMatcher(statsMatcher.RejectAll, statsMatcher.ExclusionLabels, statsMatcher.ExclusionKeys)
	metrics.SetMetricsFeature(config.FlushMosn, config.LazyFlush)

	// set metrics sample configures
	if config.SampleConfig.Type != "" {
		metrics.SetSampleType(metrics.SampleType(config.SampleConfig.Type))
	}
	if config.SampleConfig.Size > 0 {
		metrics.SetSampleSize(config.SampleConfig.Size)
	}
	if config.SampleConfig.ExpDecayAlpha > 0 {
		metrics.SetExpDecayAlpha(config.SampleConfig.ExpDecayAlpha)
	}

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

	// set ewma alpha
	if config.EWMAConfig != nil {
		switch {
		case config.EWMAConfig.Alpha > 0 && config.EWMAConfig.Alpha < 1:
			cluster.SetAlpha(config.EWMAConfig.Alpha)
		case config.EWMAConfig.Target > 0 && config.EWMAConfig.Target < 1 && config.EWMAConfig.Duration != nil:
			cluster.SetAlpha(ewma.Alpha(config.EWMAConfig.Target, config.EWMAConfig.Duration.Duration))
		default:
			log.StartLogger.Errorf("[mosn] [init metrics] invalid EWMA config, use %f as default alpha",
				cluster.GetAlpha())
		}
	}
}

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func (m *Mosn) initClusterManager() {
	log.StartLogger.Infof("[mosn start] mosn init cluster structures")
	c := m.Config

	// parse cluster all in one
	clusters, clusterMap := configmanager.ParseClusterConfig(c.ClusterManager.Clusters)
	// create cluster manager
	if mode := c.Mode(); mode == v2.Xds {
		m.Clustermanager = cluster.NewClusterManagerSingleton(nil, nil, &c.ClusterManager)
	} else {
		m.Clustermanager = cluster.NewClusterManagerSingleton(clusters, clusterMap, &c.ClusterManager)
	}

}

func (m *Mosn) initServer() {
	log.StartLogger.Infof("[mosn start] mosn init server structures")
	c := m.Config
	mode := c.Mode()

	if mode == v2.Xds {
		c.Servers = []v2.ServerConfig{
			{
				DefaultLogPath:  "stdout",
				DefaultLogLevel: "INFO",
			},
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

	// initialize the routerManager
	m.RouterManager = router.NewRouterManager()

	// TODO: Remove Servers, support only one server
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
			srv = server.NewServer(sc, cmf, m.Clustermanager)
		} else {
			//initialize server instance
			srv = server.NewServer(sc, cmf, m.Clustermanager)

			for idx := range serverConfig.Listeners {
				// parse ListenerConfig
				lc := configmanager.ParseListenerConfig(&serverConfig.Listeners[idx], m.Upgrade.InheritListeners, m.Upgrade.InheritPacketConn)
				// Note lc.FilterChains may be a nil value, and there is a check in srv.AddListener
				if _, err := srv.AddListener(lc); err != nil {
					log.StartLogger.Fatalf("[mosn] [NewMosn] AddListener error:%s", err.Error())
				}
				// deprecated: keep compatible for route config in listener's connection_manager
				deprecatedRouter, err := configmanager.ParseRouterConfiguration(&lc.FilterChains[0])
				if err != nil {
					log.StartLogger.Fatalf("[mosn] [NewMosn] compatible router: %v", err)
				}
				if deprecatedRouter.RouterConfigName != "" {
					m.RouterManager.AddOrUpdateRouters(deprecatedRouter)
				}
			}
			// Add Router Config
			for _, routerConfig := range serverConfig.Routers {
				if routerConfig.RouterConfigName != "" {
					m.RouterManager.AddOrUpdateRouters(routerConfig)
				}
			}
		}
		m.servers = append(m.servers, srv)
	}
}

// receive old connections from old mosn,
func (m *Mosn) transferConnectionHandler() error {
	// notify old mosn to transfer connection
	if _, err := m.Upgrade.ListenSockConn.Write([]byte{0}); err != nil {
		log.StartLogger.Errorf("[mosn] [NewMosn] failed to notify old mosn to transfer connection: %v, exit", err)
		return err
	}
	// wait old mosn ack
	m.Upgrade.ListenSockConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var buf [1]byte
	n, err := m.Upgrade.ListenSockConn.Read(buf[:])
	if n != 1 {
		log.StartLogger.Errorf("[mosn] [NewMosn] failed to get ack from old mosn, exit, error: %v n: %v, buf[0]: %v", err, n, buf[0])
		return errors.New("failed to get ack from old mosn")
	}

	m.Upgrade.ListenSockConn.Close()

	// receive old mosn connections
	utils.GoWithRecover(func() {
		network.TransferServer(m.servers[0].Handler())
	}, nil)

	return nil
}

func (m *Mosn) TransferConnection() (err error) {
	log.StartLogger.Infof("[mosn start] mosn transfer connections")
	// SetTransferTimeout
	network.SetTransferTimeout(server.GracefulTimeout)

	if m.Upgrade.ListenSockConn != nil {
		// start other services
		if err = store.StartService(m.Upgrade.InheritListeners); err != nil {
			log.StartLogger.Errorf("[mosn] [NewMosn] start service failed: %v, exit", err)
		}
		err = m.transferConnectionHandler()

	} else {
		// start other services
		if err = store.StartService(nil); err != nil {
			log.StartLogger.Errorf("[mosn] [NewMosn] start service failed: %v, exit", err)
		}
	}
	return
}

func (m *Mosn) CleanUpgrade() {
	log.StartLogger.Infof("[mosn start] mosn clean upgrade datas")
	//close legacy listeners
	for _, ln := range m.Upgrade.InheritListeners {
		if ln != nil {
			log.StartLogger.Infof("[mosn] [NewMosn] close useless legacy listener: %s", ln.Addr().String())
			ln.Close()
		}
	}
	//close legacy UDP listeners
	for _, ln := range m.Upgrade.InheritPacketConn {
		if ln != nil {
			log.StartLogger.Infof("[mosn] [NewMosn] close useless legacy listener: %s", ln.LocalAddr().String())
			ln.Close()
		}
	}
}

// StartXdsClient returns a ADSClient, support some extensions on it.
func (m *Mosn) StartXdsClient() *istio.ADSClient {
	c := m.Config
	log.StartLogger.Infof("[mosn start] mosn start xds client")
	xdsClient, err := istio.NewAdsClient(c)
	if err != nil {
		log.StartLogger.Errorf("start xds failed: %v", err)
	} else {
		utils.GoWithRecover(func() {
			xdsClient.Start()
		}, nil)
		m.xdsClient = xdsClient
	}
	return xdsClient

}

func (m *Mosn) HandleExtendConfig() {
	log.StartLogger.Infof("[mosn start] mosn parse extend config")
	// Notice: executed extends parsed in config order.
	for _, cfg := range m.Config.Extends {
		if err := v2.ExtendConfigParsed(cfg.Type, cfg.Config); err != nil {
			log.StartLogger.Errorf("mosn parse extend config failed, type: %s, error: %v", cfg.Type, err)
		} else {
			configmanager.SetExtend(cfg.Type, cfg.Config)
		}
	}

}

func (m *Mosn) Start() {
	log.StartLogger.Infof("[mosn start] mosn start server")
	// start dump config process
	utils.GoWithRecover(func() {
		configmanager.DumpConfigHandler()
	}, nil)

	if !m.Config.DisableUpgrade {
		stagemanager.RegisterUpgradeHandler(server.ReconfigureHandler)
		// start reconfig domain socket
		utils.GoWithRecover(func() {
			server.ReconfigureListener()
		}, nil)
	}

	// start mosn server
	for _, srv := range m.servers {

		srv := srv

		utils.GoWithRecover(func() {
			srv.Start()
		}, nil)
	}
}

// Shutdown means graceful stop servers
func (m *Mosn) Shutdown() error {
	var failed bool
	for _, srv := range m.servers {
		if err := srv.Shutdown(); err != nil {
			log.StartLogger.Errorf("[mosn shutdown] server shutdown failed: %v", err)
			failed = true
		}
	}
	if failed {
		return errors.New("failed to shutdown MOSN")
	}
	return nil
}

func (m *Mosn) Close(isUpgrade bool) {
	log.StartLogger.Infof("[mosn close] mosn stop server")

	// do not remove the pid file,
	// since the new started server may have the same pid file
	if !isUpgrade {
		pid.RemovePidFile()
		// stop reconfigure domain socket
		server.StopReconfigureHandler()

	}

	// close service
	store.CloseService()

	// stop mosn server
	for _, srv := range m.servers {
		srv.Close()
	}
	if m.xdsClient != nil {
		m.xdsClient.Stop()
	}
	if m.Clustermanager != nil {
		m.Clustermanager.Destroy()
	}
}

// transfer existing connections from old mosn,
// stage manager will stop the new mosn when return error
func (m *Mosn) InheritConnections() error {
	// transfer connection used in smooth upgrade in mosn
	err := m.TransferConnection()
	// clean upgrade finish the smooth upgrade datas
	m.CleanUpgrade()
	return err
}

func (m *Mosn) GetServer() []server.Server {
	return m.servers
}
