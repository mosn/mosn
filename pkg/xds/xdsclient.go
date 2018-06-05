package xds

import (
	"net/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v1"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v2"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"time"
)

var xdsHttpEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15003"
var xdsRpcEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15010"
var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"
var stop chan bool = make(chan bool)
var done chan bool = make(chan bool)

type xdsClient struct {
	v1 *v1.V1Client
	v2 *v2.V2Client
}

func (c *xdsClient) getConfig(callbacks *Callbacks) {

	log.DefaultLogger.Infof("start to get config from istio\n")
	chans := make([]chan bool, 2)
	chans[0] = make(chan bool)
	go c.getListenersAndRoutes(callbacks, chans[0])
	chans[1] = make(chan bool)
	go c.getClustersAndHosts(callbacks, chans[1])

	for _, ch := range chans {
		<- ch
		close(ch)
	}
}

func (c *xdsClient) getListenersAndRoutes(callbacks *Callbacks, ch chan <- bool) {
	ldsEndpoint := xdsHttpEnpoint
	rdsEnpoint := xdsHttpEnpoint
	listeners := c.v1.GetListeners(ldsEndpoint)
	if listeners == nil {
		ch <- true
		return
	}
	callbacks.OnUpdateListeners(listeners)

	for _, listener := range listeners {
		//log.DefaultLogger.Infof("listener: %+v \n", listener)
		//fmt.Printf("listener: %+v \n", listener)

		for _, filterChain := range listener.FilterChains {
			for _, filter := range filterChain.Filters {
				if filter.Config.Fields["rds"] != nil {
					rdsConfig := filter.Config.Fields["rds"].GetStructValue()
					routeConfigName := rdsConfig.Fields["route_config_name"].GetStringValue()
					route := c.v1.GetRoute(rdsEnpoint, routeConfigName)
					if route != nil {
						callbacks.OnUpdateRoutes(route)
					}
					//log.DefaultLogger.Infof("route: %+v \n", route)
					//fmt.Printf("route: %+v \n", route)
				}
			}
		}
	}
	ch <- true
}

func (c *xdsClient) getClustersAndHosts(callbacks *Callbacks, ch chan <- bool) {
	cdsEndpoint := xdsHttpEnpoint
	edsEndpoint := xdsRpcEnpoint
	clusters := c.v1.GetClusters(cdsEndpoint)
	if clusters == nil {
		ch <- true
		return
	}
	callbacks.OnUpdateClusters(clusters)

	for _, cluster := range clusters {
		//log.DefaultLogger.Infof("cluster: %+v \n", cluster)
		//fmt.Printf("cluster: %#v \n", cluster)
		if cluster.Type == xdsapi.Cluster_EDS {
			clusterName := cluster.Name
			endpoints := c.v2.GetEndpoints(edsEndpoint, clusterName)
			if endpoints != nil {
				callbacks.OnUpdateEndpoints(endpoints)
			}
		}
	}
	ch <- true
}

func Start(callbacks *Callbacks) {

	log.DefaultLogger.Infof("xdsclient start\n")
	client := xdsClient{}
	if client.v1 == nil {
		client.v1 = &v1.V1Client{&http.Client{}, serviceCluster, serviceNode}
	}
	if client.v2 == nil {
		client.v2 = &v2.V2Client{serviceCluster, serviceNode}
	}

	client.getConfig(callbacks)

	t1 := time.NewTimer(time.Second * 5)
	for {
		select {
		case <- stop:
			done <- true
		case <- t1.C:
			client.getConfig(callbacks)
			t1.Reset(time.Second * 5)
		}
	}

}

func Stop() {
	log.DefaultLogger.Infof("prepare to stop xdsclient\n")
	stop <- true
	<- done
	log.DefaultLogger.Infof("xdsclient stop\n")
}
