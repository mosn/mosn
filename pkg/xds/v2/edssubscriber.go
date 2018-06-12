package v2

import (
	//"time"
	//"golang.org/x/net/context"
	//"google.golang.org/grpc"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	google_rpc "github.com/gogo/googleapis/google/rpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *V2Client) GetEndpoints(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient, clusterName string) []*pb.ClusterLoadAssignment {
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
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return nil
	}
*/
	if streamClient == nil {
		return nil
	}
	err := streamClient.Send(&pb.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{clusterName},
		TypeUrl: "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment",
		ResponseNonce:"",
		ErrorDetail: &google_rpc.Status{

		},
		Node:&envoy_api_v2_core1.Node{
			Id:c.ServiceNode,
			Cluster:c.ServiceCluster,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return nil
	}
	r,err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
		return nil

	}
	lbAssignments := make([]*pb.ClusterLoadAssignment,0)
	for _ ,res := range r.Resources{
		lbAssignment := pb.ClusterLoadAssignment{}
		lbAssignment.Unmarshal(res.GetValue())
		lbAssignments = append(lbAssignments, &lbAssignment)
	}
	return lbAssignments
}
