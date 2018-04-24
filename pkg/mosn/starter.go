package main

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"sync"
)

func Start(c *config.MOSNConfig) {

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("mosn config : %+v\n", c)

	srvNum := len(c.Servers)
	if srvNum == 0 {
		log.Fatalln("no server found")
	}

	if c.ClusterManager.Clusters == nil || len(c.ClusterManager.Clusters) == 0 {
		log.Fatalln("no cluster found")
	}

	stopChans := make([]chan bool, srvNum)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	inheritListeners := getInheritListeners()

	for i, serverConfig := range c.Servers {
		stopChan := stopChans[i]

		//1. server config prepare
		//server config
		sc := config.ParseServerConfig(&serverConfig)

		//cluster manager filter
		cmf := &clusterManagerFilter{}

		//2. initialize server instance
		srv := server.NewServer(sc, cmf)

		//add listener
		if serverConfig.Listeners == nil || len(serverConfig.Listeners) == 0 {
			log.Fatalln("no listener found")
		}

		for _, listenerConfig := range serverConfig.Listeners {
			// network filters
			nfcf := getNetworkFilter(listenerConfig.NetworkFilters)

			//stream filters
			sfcf := getStreamFilters(listenerConfig.StreamFilters)

			srv.AddListener(config.ParseListenerConfig(&listenerConfig, inheritListeners), nfcf, sfcf)
		}

		var clusters []v2.Cluster
		clusterMap := make(map[string][]v2.Host)

		for _, cluster := range c.ClusterManager.Clusters {
			parsed := config.ParseClusterConfig(&cluster)
			clusters = append(clusters, parsed)
			clusterMap[parsed.Name] = config.ParseHostConfig(&cluster)
		}
		cmf.cccb.UpdateClusterConfig(clusters)

		for clusterName, hosts := range clusterMap {
			cmf.chcb.UpdateClusterHost(clusterName, 0, hosts)
		}

		go func() {
			srv.Start() //开启连接

			select {
			case <-stopChan:
				srv.Close()
			}
		}()
	}

	//close legacy listeners
	for _, ln := range inheritListeners {
		if !ln.Remain {
			log.Println("close useless legacy listener:", ln.Addr)
			ln.InheritListener.Close()
		}
	}

	//todo: daemon running
	wg.Wait()
}

func getNetworkFilter(configs []config.FilterConfig) types.NetworkFilterChainFactory {
	if len(configs) != 1 {
		log.Fatalln("only one network filter supported")
	}

	c := &configs[0]

	if c.Type != "proxy" {
		log.Fatalln("only proxy network filter supported")
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
			log.Fatalln("unsupport stream filter type:" + c.Type)
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

		for idx := 0; idx < count; idx++ {
			//because passed listeners fd's index starts from 3
			file := os.NewFile(uintptr(3+idx), "")
			fileListener, err := net.FileListener(file)
			if err != nil {
				log.Println("net.FileListener create err", err)
			}
			if listener, ok := fileListener.(*net.TCPListener); ok {
				listeners[idx] = &v2.ListenerConfig{Addr: listener.Addr(), InheritListener: listener}
			} else {
				log.Println("net.TCPListener cast err", err)
			}
		}
		return listeners
	}
	return nil
}
