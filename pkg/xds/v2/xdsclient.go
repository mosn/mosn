package v2

import (

	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	util "github.com/envoyproxy/go-control-plane/pkg/util"
	"net/http"
	"fmt"
	http_conn "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	//"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
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
	rdsEnpoint := xdsHttpEnpoint
	listeners := c.getListeners(ldsEndpoint)
	if listeners == nil {
		return
	}
	for _,listener := range listeners {
		fmt.Println("xds get listener : %s" , listener.String())
		for _, filterChain := range listener.FilterChains {
			for _,filter := range filterChain.Filters {
				if(filter.Name == util.HTTPConnectionManager){
					filterConfig := &http_conn.HttpConnectionManager{}
					util.StructToMessage(filter.Config,filterConfig)
					rds := filterConfig.GetRds()
					if rds != nil {
						fmt.Println("rds config name:",rds.RouteConfigName)
						ht := c.getRoute(rdsEnpoint,rds.RouteConfigName)
						for _,vh := range ht.VirtualHosts{
							fmt.Println("virtual host name:",vh.Name)
						}
					}else{
						fmt.Println("rds convert fail")
					}
				}
			}
		}
	}
}

func (c *xdsClient) getClustersAndHosts() {
	cdsEndpoint := xdsEnpoint
	edsEndpoint := xdsEnpoint
	clusters := c.getClusters(cdsEndpoint)
	for _,cluster := range clusters{
		fmt.Println("xds get cluster : %s" , cluster.String())
		if( cluster.Type == pb.Cluster_EDS ){
			lbAssignments := c.getEndpoints(edsEndpoint,cluster.Name)
			for _,lbAssignment := range lbAssignments{
				//log.DefaultLogger.Infof("get endpoint : %s",lbAssignment.String())
				fmt.Printf("xds get endpoint : %s",lbAssignment.String())
			}
		}
	}
	//eds test
	lbAssignments := c.getEndpoints(edsEndpoint,"out.s1.hw.svc.szbd-uc.uaebd.local|http-s1|version=v1")
	for _,lbAssignment := range lbAssignments{
		for _,item := range lbAssignment.Endpoints{
			item.String()
		}
		fmt.Printf("xds get endpoint : %s",lbAssignment.String())
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