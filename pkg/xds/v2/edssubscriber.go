package v2

import (
	"time"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	google_rpc "github.com/gogo/googleapis/google/rpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getEndpoints(endpoint string, cluster pb.Cluster) []pb.ClusterLoadAssignment {
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		log.DefaultLogger.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewEndpointDiscoveryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	streamClient, err := client.StreamEndpoints(ctx)
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
	}
	err = streamClient.Send(&pb.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{cluster.Name},
		TypeUrl:"",
		ResponseNonce:"",
		ErrorDetail: &google_rpc.Status{

		},
		Node:&envoy_api_v2_core1.Node{
			Id:c.serviceNode,
			Cluster:c.serviceCluster,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
	}
	r,err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get endpoints fail: %v", err)
	}
	lbAssignments := make([]pb.ClusterLoadAssignment,0)
	for _ ,res := range r.Resources{
		lbAssignment := pb.ClusterLoadAssignment{}
		lbAssignment.Unmarshal(res.GetValue())
		lbAssignments = append(lbAssignments, lbAssignment)
	}
	return lbAssignments
}
