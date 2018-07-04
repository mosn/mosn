package v2

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	//google_rpc "github.com/gogo/googleapis/google/rpc"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *V2Client) GetEndpoints(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient, clusterNames []string) []*envoy_api_v2.ClusterLoadAssignment {
	err := c.ReqEndpoints(streamClient, clusterNames)
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return nil
	}
	r, err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return nil

	}
	return c.HandleEndpointesResp(r)
}

func (c *V2Client) ReqEndpoints(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient, clusterNames []string) error {
	if streamClient == nil {
		return errors.New("stream client is nil")
	}
	err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: clusterNames,
		TypeUrl:       "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment",
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node: &envoy_api_v2_core1.Node{
			Id:      c.ServiceNode,
			Cluster: c.ServiceCluster,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return err
	}
	return nil
}

func (c *V2Client) HandleEndpointesResp(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.ClusterLoadAssignment {
	lbAssignments := make([]*envoy_api_v2.ClusterLoadAssignment, 0)
	for _, res := range resp.Resources {
		lbAssignment := envoy_api_v2.ClusterLoadAssignment{}
		lbAssignment.Unmarshal(res.GetValue())
		lbAssignments = append(lbAssignments, &lbAssignment)
	}
	return lbAssignments
}
