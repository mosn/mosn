package v2

import (
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
	"time"
)

type V2Client struct {
	ServiceCluster string
	ServiceNode string
	Config *XDSConfig
}

type XDSConfig struct {
	ADSConfig *ADSConfig
	Clusters map[string]*ClusterConfig
}

type ClusterConfig struct {
	LbPolicy xdsapi.Cluster_LbPolicy
	Address []string
	ConnectTimeout *time.Duration
}

type ADSConfig struct {
	ApiType core.ApiConfigSource_ApiType
	RefreshDelay *time.Duration
	Services []*ServiceConfig
	StreamClient *StreamClient
}

type ADSClient struct {
	adsConfig *ADSConfig
	controlChan chan int
	streamClient ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	v2Client *V2Client
}

type ServiceConfig struct {
	Timeout *time.Duration
	ClusterConfig *ClusterConfig
}

type StreamClient struct {
	Client ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	Conn *grpc.ClientConn
	Cancel context.CancelFunc
}
