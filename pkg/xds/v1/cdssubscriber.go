package v1

import (
	"fmt"
	"io/ioutil"
	"encoding/json"

	//"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getClusters(endpoint string) Clusters {
	url := c.getCDSRequest(endpoint)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		//c.logger.Errorf("couldn't get clusters: %v", err)
		fmt.Printf("couldn't get clusters: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//c.logger.Errorf("read body error when get clusters: %v", err)
		fmt.Printf("read body error when get clusters: %v\n", err)
		return nil
	}
	return c.parseClusters(body)
}

func (c *xdsClient) parseClusters(body []byte) Clusters {
	res := ClusterManager{}
	err := json.Unmarshal(body, &res)
	if err != nil {
		//c.logger.Errorf("fail to unmarshal clusters config: %v", err)
		fmt.Printf("fail to unmarshal clusters config: %v\n", err)
	}
	//fmt.Printf("cluster name: %s", len(res.Clusters[0].Name))
	return res.Clusters
}

func (c *xdsClient) getCDSRequest(endpoint string) string {
	return fmt.Sprintf("http://%s/v1/clusters/%s/%s", endpoint, c.serviceCluster, c.serviceNode)
}