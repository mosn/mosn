package v2

import (
	//"time"
	//"google.golang.org/grpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	//"golang.org/x/net/context"
	google_rpc "github.com/gogo/googleapis/google/rpc"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func (c *V2Client) GetClusters(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient) []*envoy_api_v2.Cluster {
		// Set up a connection to the server.
/*
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		log.DefaultLogger.Fatalf("did not connect: %v", err)
		return nil
	}
	defer conn.Close()
	client := ads.NewAggregatedDiscoveryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	streamClient, err := client.StreamAggregatedResources(ctx)
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
		return nil
	}
*/
	if streamClient == nil {
		return nil
	}
	err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{},
		TypeUrl:"type.googleapis.com/envoy.api.v2.Cluster",
		ResponseNonce:"",
		ErrorDetail: &google_rpc.Status{

		},
		Node:&envoy_api_v2_core1.Node{
			Id:c.ServiceNode,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
		return nil
	}
	r,err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
		return nil
	}
	clusters := make([]*envoy_api_v2.Cluster,0)
	for _ ,res := range r.Resources{
		cluster := envoy_api_v2.Cluster{}
		cluster.Unmarshal(res.GetValue())
		clusters = append(clusters,&cluster)
	}
	return clusters
}

func (c *V2Client) ReqClusters(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient) {
		if streamClient == nil {
		return
	}
	err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{},
		TypeUrl:"type.googleapis.com/envoy.api.v2.Cluster",
		ResponseNonce:"",
		ErrorDetail: &google_rpc.Status{

		},
		Node:&envoy_api_v2_core1.Node{
			Id:c.ServiceNode,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
		return
	}
}

func (c *V2Client) HandleClustersResp(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.Cluster{
	clusters := make([]*envoy_api_v2.Cluster,0)
	for _ ,res := range resp.Resources{
		cluster := envoy_api_v2.Cluster{}
		cluster.Unmarshal(res.GetValue())
		clusters = append(clusters,&cluster)
	}
	return clusters
}
