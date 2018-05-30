package v1

import (
	"fmt"
	"io/ioutil"
	"encoding/json"

	//"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getHosts(endpoint, serviceName string) *ServiceHosts {
	url := c.getSDSResquest(endpoint, serviceName)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		//c.logger.Errorf("couldn't get hosts: %v", err)
		fmt.Printf("couldn't get hosts: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//c.logger.Errorf("read body error when get hosts: %v", err)
		fmt.Printf("read body error when get hosts: %v\n", err)
		return nil
	}
	return c.parseHosts(body)
}

func (c *xdsClient) parseHosts(body []byte) *ServiceHosts {
	res := ServiceHosts{}
	err := json.Unmarshal(body, &res)
	if err != nil {
		//c.logger.Errorf("fail to unmarshal hosts config: %v", err)
		fmt.Printf("fail to unmarshal hosts config: %v\n", err)
	}
	return &res
}

func (c *xdsClient) getSDSResquest(endpoint, serviceName string) string {
	return fmt.Sprintf("http://%s/v1/registration/%s", endpoint, serviceName)
}
