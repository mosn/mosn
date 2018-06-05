package v1

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	google_protobuf "github.com/gogo/protobuf/types"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func (c *xdsClient) getClusters(endpoint string) Clusters {
	url := c.getCDSRequest(endpoint)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.DefaultLogger.Errorf("couldn't get clusters: %v", err)
		//fmt.Printf("couldn't get clusters: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Errorf("read body error when get clusters: %v", err)
		//fmt.Printf("read body error when get clusters: %v\n", err)
		return nil
	}
	return c.parseClusters(body)
}

func (c *xdsClient) parseClusters(body []byte) Clusters {
	res := ClusterManager{}
	err := json.Unmarshal(body, &res)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal clusters config: %v", err)
		//fmt.Printf("fail to unmarshal clusters config: %v\n", err)
		return nil
	}
	//fmt.Printf("cluster name: %s", len(res.Clusters[0].Name))

	clusters := make([]*xdsapi.Cluster, 0, len(res.ClustersV1))
	for _, clusterV1 := range res.ClustersV1 {
		clusterV2 := xdsapi.Cluster{}
		clusterV2.Name = clusterV1.Name
		clusterType := clusterV1.Type

		switch clusterType {
		case "static":
			clusterV2.Type = xdsapi.Cluster_STATIC
			err = setDnsHost(clusterV1, true, true, &clusterV2)
			if err != nil {
				log.DefaultLogger.Errorf("fail to set dns host: %v", err)
				//fmt.Printf("fail to set dns host: %v\n", err)
				return nil
			}
		case "strict_dns":
			clusterV2.Type = xdsapi.Cluster_STRICT_DNS
			setDnsHost(clusterV1, true, false, &clusterV2)
			if err != nil {
				log.DefaultLogger.Errorf("fail to set dns host: %v", err)
				//fmt.Printf("fail to set dns host: %v\n", err)
				return nil
			}
		case "logical_dns":
			clusterV2.Type = xdsapi.Cluster_LOGICAL_DNS
			setDnsHost(clusterV1, true, false, &clusterV2)
			if err != nil {
				log.DefaultLogger.Errorf("fail to set dns host: %v", err)
				//fmt.Printf("fail to set dns host: %v\n", err)
				return nil
			}
		case "original_dst":
			if len(clusterV1.Hosts) != 0 {
				log.DefaultLogger.Errorf("original_dst clusters must have no hosts configured")
				//fmt.Print("original_dst clusters must have no hosts configured\n")
				return nil
			}
			clusterV2.Type = xdsapi.Cluster_ORIGINAL_DST
		case "sds":
			clusterV2.Type = xdsapi.Cluster_EDS
			clusterV2.EdsClusterConfig = &xdsapi.Cluster_EdsClusterConfig{}
			clusterV2.EdsClusterConfig.EdsConfig = &core.ConfigSource{} //translateEdsConfig()
			clusterV2.EdsClusterConfig.ServiceName = clusterV1.ServiceName
		}

		var nanoseconds int64 = clusterV1.ConnectTimeoutMs * int64(time.Millisecond)
		clusterV2.ConnectTimeout = time.Duration(nanoseconds)
		clusterV2.PerConnectionBufferLimitBytes = &google_protobuf.UInt32Value{uint32(clusterV1.PerConnectionBufferLimitBytes)}

		lbType := clusterV1.LbType
		switch lbType {
		case "round_robin":
			clusterV2.LbPolicy = xdsapi.Cluster_ROUND_ROBIN
		case "least_request":
			clusterV2.LbPolicy = xdsapi.Cluster_LEAST_REQUEST
		case "random":
			clusterV2.LbPolicy = xdsapi.Cluster_RANDOM
		case "original_dst_lb":
			clusterV2.LbPolicy = xdsapi.Cluster_ORIGINAL_DST_LB
		default:
			log.DefaultLogger.Errorf("unsupport LbPolicy %s", lbType)
			//fmt.Printf("unsupport LbPolicy %s\n", lbType)
			return nil
		}
		if clusterV1.CircuitBreaker != nil {
			clusterV2.CircuitBreakers = &cluster.CircuitBreakers{}
			err = translateCircuitBreaker(clusterV1.CircuitBreaker, clusterV2.CircuitBreakers)
			if err != nil {
				log.DefaultLogger.Errorf("fail to translate circuit breaker: %v", err)
				//fmt.Printf("fail to translate circuit breaker: %v", err)
				return nil
			}
		}
		//clusterV2.HealthChecks = translateHealthChecks(clusterV1.HealthChecks)
		if clusterV1.OutlierDetection != nil {
			clusterV2.OutlierDetection = &cluster.OutlierDetection{}
			err = translateOutlierDetection(clusterV1.OutlierDetection, clusterV2.OutlierDetection)
			if err != nil {
				log.DefaultLogger.Errorf("fail to translate outlier detection: %v", err)
				//fmt.Printf("fail to translate outlier detection: %v", err)
				return nil
			}
		}
		clusters = append(clusters, &clusterV2)
	}

	return clusters
}

func (c *xdsClient) getCDSRequest(endpoint string) string {
	return fmt.Sprintf("http://%s/v1/clusters/%s/%s", endpoint, c.serviceCluster, c.serviceNode)
}