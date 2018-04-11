package main

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"log"
	"net/http"
	"time"
)

func Start(c *config.MOSNConfig) {
	fmt.Printf("mosn config : %+v\n", c)

	srvNum := len(c.Servers)
	if srvNum == 0 {
		log.Fatal("no server found")
	}

	if c.ClusterManager.Clusters == nil || len(c.ClusterManager.Clusters) == 0 {
		log.Fatal("no cluster found")
	}

	stopChans := make([]chan bool, srvNum)

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9099", nil)
	}()

	for i, serverConfig := range c.Servers {
		stopChan := stopChans[i]
		//start server
		go func() {
			//1. server config prepare
			//server config
			sc := config.ParseServerConfig(&serverConfig)

			// network filters
			nfcf := getNetworkFilter(serverConfig.NetworkFilters)

			//stream filters
			sfcf := getStreamFilters(serverConfig.StreamFilters)

			//cluster manager filter
			cmf := &clusterManagerFilter{}

			//2. initialize server instance
			srv := server.NewServer(sc, nfcf, sfcf, cmf)

			//add listener
			if serverConfig.Listeners == nil || len(serverConfig.Listeners) == 0 {
				log.Fatal("no listener found")
			}

			for _, listenerConfig := range serverConfig.Listeners {
				srv.AddListener(config.ParseListenerConfig(&listenerConfig))
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

			srv.Start() //开启连接

			fmt.Println("[MAIN]mosn server started..")

			select {
			case <-stopChan:
				srv.Close()
			}
		}()
	}

	select {
	case <-time.After(time.Second * 120):
		stopChans[0] <- true
		fmt.Println("[MAIN]closing..")
	}

}

func getNetworkFilter(configs []config.FilterConfig) server.NetworkFilterChainFactory {
	if len(configs) != 1 {
		log.Fatal("only one network filter supported")
	}

	c := &configs[0]

	if c.Type != "proxy" {
		log.Fatal("only proxy network filter supported")
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
			log.Fatal("unsupport stream filter type:" + c.Type)
		}

	}
	return factories
}

type clusterManagerFilter struct {
	cccb server.ClusterConfigFactoryCb
	chcb server.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb server.ClusterConfigFactoryCb, chcb server.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}
