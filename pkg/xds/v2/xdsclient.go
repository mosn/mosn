package v2

import (

	//"github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	//util "github.com/envoyproxy/go-control-plane/pkg/util"
	"net/http"
	"fmt"
)

var xdsEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15010"
var xdsHttpEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15003"
var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"

func NewClient(serviceCluster, serviceNode string) *xdsClient {
	return &xdsClient{&http.Client{},serviceCluster, serviceNode}
}

func (c *xdsClient) getConfig() {
	//go func() {
		c.getListenersAndRoutes()
	//}()

	//go func() {
		c.getClustersAndHosts()
	//}()

}

func (c *xdsClient) getListenersAndRoutes() {
	ldsEndpoint := xdsEnpoint
	//rdsEnpoint := xdsHttpEnpoint
	listeners := c.getListeners(ldsEndpoint)
	if listeners == nil {
		return
	}
	for _,listener := range listeners {
		fmt.Println("xds get listener : %s" , listener.String())
		//for _, filterChain := range listener.FilterChains {
		//	for _,filter := range filterChain.Filters {
		//		if(filter.Name == util.HTTPConnectionManager){
		//			filterConfig := &HttpConnectionManager{}
		//			util.StructToMessage(filter.Config,filterConfig)
		//			rds ,err := filterConfig.RouteSpecifier.(*http_conn.HttpConnectionManager_Rds)
		//			if !err {
		//				c.getRoute(rdsEnpoint,rds.Rds.RouteConfigName)
		//			}
		//		}
		//	}
		//}
	}
}

func (c *xdsClient) getClustersAndHosts() {
	cdsEndpoint := xdsEnpoint
	edsEndpoint := xdsEnpoint
	clusters := c.getClusters(cdsEndpoint)
	for _,cluster := range clusters{
		fmt.Println("xds get cluster : %s" , cluster.String())
		if( cluster.Type == pb.Cluster_EDS ){
			lbAssignments := c.getEndpoints(edsEndpoint,cluster)
			for _,lbAssignment := range lbAssignments{
				//log.DefaultLogger.Infof("get endpoint : %s",lbAssignment.String())
				fmt.Printf("xds get endpoint : %s",lbAssignment.String())
			}
		}
	}

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