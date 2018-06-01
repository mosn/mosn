package v2

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	v1 "gitlab.alipay-inc.com/afe/mosn/pkg/xds/v1"

	//"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getRoute(endpoint, routeConfigName string) *v1.HTTPRouteConfig {
	url := c.getRDSResquest(endpoint, routeConfigName)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		//c.logger.Errorf("couldn't get routes: %v", err)
		fmt.Printf("couldn't get routes: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//c.logger.Errorf("read body error when get routes: %v", err)
		fmt.Printf("read body error when get routes: %v\n", err)
		return nil
	}
	return c.parseRoutes(body)
}

func (c *xdsClient) parseRoutes(body []byte) *v1.HTTPRouteConfig {
	var res v1.HTTPRouteConfig
	err := json.Unmarshal(body, &res)
	if err != nil {
		//c.logger.Errorf("fail to unmarshal route config: %v", err)
		fmt.Printf("fail to unmarshal route config: %v\n", err)
	}
	//fmt.Printf("virtual host name: %s", res.VirtualHosts[0].Name)
	return &res
}

func (c *xdsClient) getRDSResquest(endpoint, routeConfigName string) string {
	return fmt.Sprintf("http://%s/v1/routes/%s/%s/%s", endpoint, routeConfigName, c.serviceCluster, c.serviceNode)
}
