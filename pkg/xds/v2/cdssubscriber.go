package v2

import (
	"time"
	"google.golang.org/grpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"golang.org/x/net/context"
	google_rpc "github.com/gogo/googleapis/google/rpc"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"fmt"
)

func (c *xdsClient) getClusters(endpoint string) []pb.Cluster {
		// Set up a connection to the server.
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		log.DefaultLogger.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewClusterDiscoveryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	streamClient, err := client.StreamClusters(ctx)
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
	}
	err = streamClient.Send(&pb.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{},
		TypeUrl:"",
		ResponseNonce:"",
		ErrorDetail: &google_rpc.Status{

		},
		Node:&envoy_api_v2_core1.Node{
			Id:c.serviceNode,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
	}
	r,err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get clusters fail: %v", err)
	}
	clusters := make([]pb.Cluster,0)
	for _ ,res := range r.Resources{
		cluster := pb.Cluster{}
		cluster.Unmarshal(res.GetValue())
		clusters = append(clusters,cluster)
	}
	return clusters
}

