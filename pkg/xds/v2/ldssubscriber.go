package v2

import (
	//"time"
	//"golang.org/x/net/context"
	//"google.golang.org/grpc"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_api_v2_core1 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	//google_rpc "github.com/gogo/googleapis/google/rpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"errors"
)

func (c *V2Client) GetListeners(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient) []*envoy_api_v2.Listener{
	err := c.ReqListeners(streamClient)
	if err != nil {
		log.DefaultLogger.Fatalf("get listener fail: %v", err)
		return nil
	}
	r,err := streamClient.Recv()
	if err != nil {
		log.DefaultLogger.Fatalf("get listener fail: %v", err)
		return nil
	}
	return c.HandleListersResp(r)
}


func (c *V2Client) ReqListeners(streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient) error{
	if streamClient == nil {
		return errors.New("stream client is nil")
	}
	err := streamClient.Send(&envoy_api_v2.DiscoveryRequest{
		VersionInfo:"",
		ResourceNames: []string{},
		TypeUrl:"type.googleapis.com/envoy.api.v2.Listener",
		ResponseNonce:"",
		ErrorDetail: nil,
		Node:&envoy_api_v2_core1.Node{
			Id:c.ServiceNode,
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("get listener fail: %v", err)
		return err
	}
	return nil
}

func (c *V2Client) HandleListersResp(resp *envoy_api_v2.DiscoveryResponse) []*envoy_api_v2.Listener{
	listeners := make([]*envoy_api_v2.Listener,0)
	for _ ,res := range resp.Resources{
		listener := envoy_api_v2.Listener{}
		listener.Unmarshal(res.GetValue())
		listeners = append(listeners, &listener)
	}
	return listeners
}

