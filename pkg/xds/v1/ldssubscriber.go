package v1

import (
	"fmt"
	"io/ioutil"
	"encoding/json"

	//"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getListeners(endpoint string) Listeners{
	url := c.getLDSResquest(endpoint)
	fmt.Println(url)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		//c.logger.Errorf("couldn't get listeners: %v", err)
		fmt.Printf("couldn't get listeners: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//c.logger.Errorf("read body error when get listeners: %v", err)
		fmt.Printf("read body error when get listeners: %v\n", err)
		return nil
	}
	return c.parseListeners(body)
}

func (c *xdsClient) parseListeners(body []byte) Listeners {
	var res ldsResponse

	err := json.Unmarshal(body, &res)
	if err != nil {
		//c.logger.Errorf("read body error when get listeners: %v", err)
		fmt.Printf("read body error when get listeners: %v\n", err)
		return nil
	}

	for _, listener := range res.Listeners {
		//fmt.Printf("listener name: %s\n", listener.Name)
		if len(listener.Filters) < 1 {
			//c.logger.Errorf("network filter not found for listener: %#v", listener)
			//fmt.Printf("network filter not found for listener: %#v\n", listener)
			continue
		}
		filter := listener.Filters[0]
		if filter.Name == "http_connection_manager" {
			var httpConfig HTTPFilterConfig
			err := json.Unmarshal(filter.Config, &httpConfig)
			if err != nil {
				//c.logger.Errorf("fail to unmarshal http_connection_manager filter config: %v", err)
				fmt.Printf("fail to unmarshal http_connection_manager filter config: %v\n", err)
				continue
			}
			filter.HTTPFilterConfig = &httpConfig

		} else if filter.Name == "tcp_proxy" {
			var tcpConfig TCPProxyFilterConfig
			err := json.Unmarshal(filter.Config, &tcpConfig)
			if err != nil {
				//c.logger.Errorf("fail to unmarshal http_connection_manager filter config: %v", err)
				fmt.Printf("fail to unmarshal http_connection_manager filter config: %v\n", err)
				continue
			}
			filter.TCPProxyFilterConfig = &tcpConfig
		}
	}

	return res.Listeners
}

func (c *xdsClient) getLDSResquest(endpoint string) string {
	return fmt.Sprintf("http://%s/v1/listeners/%s/%s", endpoint, c.serviceCluster, c.serviceNode)
}