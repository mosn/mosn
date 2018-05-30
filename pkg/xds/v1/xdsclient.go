package v1

import (
	"net/http"

	//"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var xdsEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15003"
var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"

func NewClient(serviceCluster, serviceNode string) *xdsClient {
	return &xdsClient{&http.Client{}, serviceCluster, serviceNode}
}

func (c *xdsClient) getConfig() {
	go func() {
		c.getListenersAndRoutes()
	}()

	go func() {
		c.getClustersAndHosts()
	}()

}

func (c *xdsClient) getListenersAndRoutes() {
	ldsEndpoint := xdsEnpoint
	rdsEnpoint := xdsEnpoint
	listeners := c.getListeners(ldsEndpoint)
	if listeners == nil {
		return
	}
	for _,listener := range listeners {
		for _, filters := range listener.Filters {
			if filters.HTTPFilterConfig != nil && filters.HTTPFilterConfig.RDS != nil{
				routeConfigName := filters.HTTPFilterConfig.RDS.RouteConfigName
				c.getRoute(rdsEnpoint, routeConfigName)
			}
		}
	}
}

func (c *xdsClient) getClustersAndHosts() {
	cdsEndpoint := xdsEnpoint
	c.getClusters(cdsEndpoint)
}

func Start() {
/*
	logPath := "/tmp/xdsclient.log" //MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
	logLevel := INFO
	logger, err := log.NewLogger(logPath, log.LogLevel(logLevel))
	if err != nil {
		return
	}
*/
	client := NewClient(serviceCluster, serviceNode)
	client.getConfig()
}

func Stop() {

}