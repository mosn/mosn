package v1

import (
	"net/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"time"
)

var xdsEnpoint string =  "istio-pilot.istio-system-beta.svc.szbd-uc.uaebd.local:15003"
var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"
var stop chan bool = make(chan bool)
var done chan bool = make(chan bool)

func NewClient(serviceCluster, serviceNode string) *xdsClient {
	return &xdsClient{&http.Client{}, serviceCluster, serviceNode}
}

func (c *xdsClient) getConfig() {

	log.DefaultLogger.Infof("start to get config from istio\n")
	chans := make([]chan bool, 2)
	chans[0] = make(chan bool)
	go c.getListenersAndRoutes(chans[0])
	chans[1] = make(chan bool)
	go c.getClustersAndHosts(chans[1])

	for _, ch := range chans {
		<- ch
		close(ch)
	}
}

func (c *xdsClient) getListenersAndRoutes(ch chan <- bool) {
	ldsEndpoint := xdsEnpoint
	rdsEnpoint := xdsEnpoint
	listeners := c.getListeners(ldsEndpoint)
	if listeners == nil {
		ch <- true
		return
	}

	for _,listener := range listeners {
		log.DefaultLogger.Infof("listener: %+v \n", listener)
		//fmt.Printf("listener: %+v \n", listener)

		for _, filterChain := range listener.FilterChains {
			for _, filter := range filterChain.Filters {
				if filter.Config.Fields["value"] != nil {
					config := filter.Config.Fields["value"].GetStructValue()
					if config.Fields["rds"] != nil {
						rdsConfig := config.Fields["rds"].GetStructValue()
						routeConfigName := rdsConfig.Fields["route_config_name"].GetStringValue()
						route := c.getRoute(rdsEnpoint, routeConfigName)
						log.DefaultLogger.Infof("route: %+v \n", route)
						//fmt.Printf("route: %+v \n", route)
					}
				}
			}
		}
	}
	ch <- true
}

func (c *xdsClient) getClustersAndHosts(ch chan <- bool) {
	cdsEndpoint := xdsEnpoint
	clusters := c.getClusters(cdsEndpoint)

	for _, cluster := range clusters {
		log.DefaultLogger.Infof("cluster: %+v \n", cluster)
		//fmt.Printf("cluster: %#v \n", cluster)
	}
	ch <- true
}

func Start() {

	logPath := "/tmp/xdsclient.log" //MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
	logLevel := log.INFO
	err := log.InitDefaultLogger(logPath, log.LogLevel(logLevel))
	if err != nil {
		return
	}
	
	log.DefaultLogger.Infof("xdsclient start\n")
	client := NewClient(serviceCluster, serviceNode)
	client.getConfig()

	t1 := time.NewTimer(time.Second * 5)
	for {
		select {
		case <- stop:
			done <- true
		case <- t1.C:
			client.getConfig()
			t1.Reset(time.Second * 5)
		}
	}
	log.CloseAll()

}

func Stop() {
	log.DefaultLogger.Infof("prepare to stop xdsclient\n")
	stop <- true
	<- done
	log.DefaultLogger.Infof("xdsclient stop\n")
}