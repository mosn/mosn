package v1

import (
	"fmt"
	"io/ioutil"
	"encoding/json"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	google_protobuf "github.com/gogo/protobuf/types"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

func (c *xdsClient) getRoute(endpoint, routeConfigName string) *xdsapi.RouteConfiguration {
	url := c.getRDSResquest(endpoint, routeConfigName)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.DefaultLogger.Errorf("couldn't get routes: %v", err)
		//fmt.Printf("couldn't get routes: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Errorf("read body error when get routes: %v", err)
		//fmt.Printf("read body error when get routes: %v\n", err)
		return nil
	}
	return c.parseRoutes(body)
}

func (c *xdsClient) parseRoutes(body []byte) *xdsapi.RouteConfiguration {
	var res HTTPRouteConfig
	err := json.Unmarshal(body, &res)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal route config: %v", err)
		//fmt.Printf("fail to unmarshal route config: %v\n", err)
		return nil
	}

	routesV2 := xdsapi.RouteConfiguration{}
	routesV2.VirtualHosts = make([]envoy_api_v2_route.VirtualHost, 0, len(res.VirtualHosts))
	for _, virtualHostV1 := range res.VirtualHosts {
		virtualHostV2 := envoy_api_v2_route.VirtualHost{}
		err := translateVirtualHost(virtualHostV1, &virtualHostV2)
		if err != nil {
			log.DefaultLogger.Errorf("fail to translate virtual hots: %v", err)
			//fmt.Printf("fail to translate virtual hots: %v\n", err)
			return nil
		}
		routesV2.VirtualHosts = append(routesV2.VirtualHosts, virtualHostV2)
	}
	routesV2.ValidateClusters = &google_protobuf.BoolValue{res.ValidateClusters}

	return &routesV2
}

func (c *xdsClient) getRDSResquest(endpoint, routeConfigName string) string {
	return fmt.Sprintf("http://%s/v1/routes/%s/%s/%s", endpoint, routeConfigName, c.serviceCluster, c.serviceNode)
}
