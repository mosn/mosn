package xds

import (
	"net/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v1"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v2"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"time"
	"errors"
)

var xdsHttpEnpoint string =  "istio-pilot.istio-system.svc.szbd-uc.uaebd.local:15003"
var xdsRpcEnpoint string =  "istio-pilot.istio-system.svc.szbd-uc.uaebd.local:15010"
var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"
var warmuped chan bool = make(chan bool)
var stopping chan bool = make(chan bool)
var stopped chan bool = make(chan bool)

type xdsClient struct {
	v1 *v1.V1Client
	v2 *v2.V2Client
}

func (c *xdsClient) getConfig(config *Config) (err error){

	log.DefaultLogger.Infof("start to get config from istio\n")
	chans := make([]chan bool, 2)
	chans[0] = make(chan bool)
	go c.getListenersAndRoutes(config, chans[0])
	chans[1] = make(chan bool)
	go c.getClustersAndHosts(config, chans[1])

	success := true
	for _, ch := range chans {
		v := <- ch
		success = success && v
		close(ch)
	}
	if !success {
		log.DefaultLogger.Infof("fail to get config from istio\n")
		err = errors.New("fail to get config from istio")
		return err
	}
	return nil
}

func (c *xdsClient) getListenersAndRoutes(config *Config, ch chan <- bool) {
	ldsEndpoint := xdsRpcEnpoint
	rdsEnpoint := xdsHttpEnpoint
	listeners := c.v2.GetListeners(ldsEndpoint)
	if listeners == nil {
		ch <- false
		return
	}
	config.OnUpdateListeners(listeners)

	success := true
	for _, listener := range listeners {
		//log.DefaultLogger.Infof("listener: %+v \n", listener)
		//fmt.Printf("listener: %+v \n", listener)
		for _, filterChain := range listener.FilterChains {
			for _, filter := range filterChain.Filters {
				if filter.Config.Fields["rds"] != nil {
					rdsConfig := filter.Config.Fields["rds"].GetStructValue()
					routeConfigName := rdsConfig.Fields["route_config_name"].GetStringValue()
					route := c.v1.GetRoute(rdsEnpoint, routeConfigName)
					if route == nil {
						success = false
					}
					config.OnUpdateRoutes(route)

					//log.DefaultLogger.Infof("route: %+v \n", route)
					//fmt.Printf("route: %+v \n", route)
				}
			}
		}
	}

	ch <- success
}

func (c *xdsClient) getClustersAndHosts(config *Config, ch chan <- bool) {
	cdsEndpoint := xdsHttpEnpoint
	edsEndpoint := xdsRpcEnpoint
	clusters := c.v1.GetClusters(cdsEndpoint)
	if clusters == nil {
		ch <- false
		return
	}
	config.OnUpdateClusters(clusters)

	success := true
	for _, cluster := range clusters {
		//log.DefaultLogger.Infof("cluster: %+v \n", cluster)
		//fmt.Printf("cluster: %#v \n", cluster)
		if cluster.Type == xdsapi.Cluster_EDS {
			clusterName := cluster.Name
			endpoints := c.v2.GetEndpoints(edsEndpoint, clusterName)
			if endpoints == nil {
				success = false
			}
			config.OnUpdateEndpoints(endpoints)
		}
	}
	ch <- success
}

func Start(config *Config) {

	log.DefaultLogger.Infof("xdsclient start\n")
	client := xdsClient{}
	if client.v1 == nil {
		client.v1 = &v1.V1Client{&http.Client{}, serviceCluster, serviceNode}
	}
	if client.v2 == nil {
		client.v2 = &v2.V2Client{serviceCluster, serviceNode}
	}

	for {
		err := client.getConfig(config)
		if err == nil {
			break
		}
	}
	warmuped <- true

	t1 := time.NewTimer(time.Second * 5)
	for {
		select {
		case <- stopping:
			stopped <- true
		case <- t1.C:
			client.getConfig(config)
			t1.Reset(time.Second * 5)
		}
	}

}

func Stop() {
	log.DefaultLogger.Infof("prepare to stop xdsclient\n")
	stopping <- true
	<- stopped
	log.DefaultLogger.Infof("xdsclient stop\n")
}

func WaitForWarmUp() {
	<- warmuped
}
