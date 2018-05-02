package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"

	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg"
)

func Start(c *config.MOSNConfig) {
	log.StartLogger.Infof("start by config : %+v", c)

	runtime.GOMAXPROCS(runtime.NumCPU())

	srvNum := len(c.Servers)
	if srvNum == 0 {
		log.StartLogger.Fatalln("no server found")
	} else if srvNum > 1 {
		log.StartLogger.Fatalln("multiple server not supported yet, got ", srvNum)
	}

	if c.ClusterManager.Clusters == nil || len(c.ClusterManager.Clusters) == 0 {
		log.StartLogger.Fatalln("no cluster found")
	}

	stopChans := make([]chan bool, srvNum)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	//get inherit fds
	inheritListeners := getInheritListeners()

	for i, serverConfig := range c.Servers {
		stopChan := stopChans[i]

		//1. server config prepare
		//server config
		sc := config.ParseServerConfig(&serverConfig)

		//cluster manager filter
		cmf := &clusterManagerFilter{}
		var clusters []v2.Cluster
		clusterMap := make(map[string][]v2.Host)

		for _, cluster := range c.ClusterManager.Clusters {
			parsed := config.ParseClusterConfig(&cluster)
			clusters = append(clusters, parsed)
			clusterMap[parsed.Name] = config.ParseHostConfig(&cluster)
		}
		//create cluster manager
		cm := cluster.NewClusterManager(nil,clusters,clusterMap)
		//initialize server instance
		srv := server.NewServer(sc, cmf, cm)

		//add listener
		if serverConfig.Listeners == nil || len(serverConfig.Listeners) == 0 {
			log.StartLogger.Fatalln("no listener found")
		}

		for _, listenerConfig := range serverConfig.Listeners {
			// network filters
			nfcf := getNetworkFilter(listenerConfig.NetworkFilters)

			//stream filters
			sfcf := getStreamFilters(listenerConfig.StreamFilters)

			srv.AddListener(config.ParseListenerConfig(&listenerConfig, inheritListeners), nfcf, sfcf)
		}

		go func() {
			srv.Start()
			select {
			case <-stopChan:
				srv.Close()
			}
		}()
	}

	//close legacy listeners
	for _, ln := range inheritListeners {
		if !ln.Remain {
			log.StartLogger.Println("close useless legacy listener:", ln.Addr)
			ln.InheritListener.Close()
		}
	}

	//todo: daemon running
	wg.Wait()
}

func getNetworkFilter(configs []config.FilterConfig) types.NetworkFilterChainFactory {
	if len(configs) != 1 {
		log.StartLogger.Fatalln("only one network filter supported, but got ", len(configs))
	}

	c := &configs[0]

	if c.Type != "proxy" {
		log.StartLogger.Fatalln("only proxy network filter supported, but got ", c.Type)
	}

	return &proxy.GenericProxyFilterConfigFactory{
		Proxy: config.ParseProxyFilter(c),
	}
}

func getStreamFilters(configs []config.FilterConfig) []types.StreamFilterChainFactory {
	var factories []types.StreamFilterChainFactory

	for _, c := range configs {
		switch c.Type {
		case "fault_inject":
			factories = append(factories, &faultinject.FaultInjectFilterConfigFactory{
				FaultInject: config.ParseFaultInjectFilter(c.Config),
			})
		case "healthcheck":
			factories = append(factories, &sofarpc.HealthCheckFilterConfigFactory{
				FilterConfig: config.ParseHealthcheckFilter(c.Config),
			})
		default:
			log.StartLogger.Fatalln("unsupport stream filter type: ", c.Type)
		}

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
