package v2

import (
	"runtime/debug"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sofastack.io/sofa-mosn/pkg/log"
)

type SdsSubscriber struct {
	provider           SecretProvider
	reqQueue           chan string
	sdsConfig          *core.ConfigSource
	sdsStreamClient    *SdsStreamClient
	sendStopChannel    chan int
	receiveStopChannel chan int
	serviceNode        string
	serviceCluster     string
}

type SdsStreamClient struct {
	sdsStreamConfig     *SdsStreamConfig
	conn                *grpc.ClientConn
	cancel              context.CancelFunc
	streamSecretsClient v2.SecretDiscoveryService_StreamSecretsClient
}

type SdsStreamConfig struct {
	sdsUdsPath string
	statPrefix string
}

func NewSdsSubscriber(provider SecretProvider, sdsConfig *core.ConfigSource, serviceNode string, serviceCluster string) *SdsSubscriber {
	return &SdsSubscriber{
		provider:           provider,
		reqQueue:           make(chan string, 10240),
		sdsConfig:          sdsConfig,
		sendStopChannel:    make(chan int),
		receiveStopChannel: make(chan int),
		serviceNode:        serviceNode,
		serviceCluster:     serviceCluster,
		sdsStreamClient:    nil,
	}
}

func (subscribe *SdsSubscriber) Start() error {
	sdsStreamConfig := subscribe.convertSdsConfig(subscribe.sdsConfig)
	streamClient, err := subscribe.getSdsStreamClient(sdsStreamConfig)
	if err != nil {
		return err
	}
	subscribe.sdsStreamClient = streamClient
	go subscribe.sendRequestLoop(subscribe.sdsStreamClient)
	go subscribe.receiveResponseLoop(subscribe.sdsStreamClient)
	return nil
}

func (subscribe *SdsSubscriber) Stop() {
	subscribe.sendStopChannel <- 0
	subscribe.receiveStopChannel <- 0
}

func (subscribe *SdsSubscriber) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

func (subscribe *SdsSubscriber) convertSdsConfig(sdsConfig *core.ConfigSource) *SdsStreamConfig {
	sdsStreamConfig := &SdsStreamConfig{}
	if apiConfig, ok := subscribe.sdsConfig.ConfigSourceSpecifier.(*core.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == core.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Errorf("[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return nil
			}
			if grpcConfig, ok := grpcService[0].TargetSpecifier.(*core.GrpcService_GoogleGrpc_); ok {
				sdsStreamConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
				sdsStreamConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
			}
		}
	}
	return sdsStreamConfig
}

func (subscribe *SdsSubscriber) sendRequestLoop(sdsStreamClient *SdsStreamClient) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[xds] [sds subscriber] panic %v\n%s", r, string(debug.Stack()))
		}
	}()
	for {
		select {
		case <-subscribe.sendStopChannel:
			return
		case name := <-subscribe.reqQueue:
			discoveryReq := &xdsapi.DiscoveryRequest{
				ResourceNames: []string{name},
				Node: &core.Node{
					Id: subscribe.serviceNode,
				},
			}
			err := subscribe.sendRequest(discoveryReq)
			if err != nil {
				log.DefaultLogger.Errorf("[xds] [sds subscriber] send sds request fail , resource name = %v", name)
				subscribe.reconnect()
			}
		}
	}
}

func (subscribe *SdsSubscriber) receiveResponseLoop(sdsStreamClient *SdsStreamClient) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[xds] [sds subscriber] panic %v\n%s", r, string(debug.Stack()))
		}
	}()
	for {
		select {
		case <-subscribe.receiveStopChannel:
			return
		default:
			if subscribe.sdsStreamClient == nil {
				log.DefaultLogger.Warnf("[xds] [sds subscriber] stream client closed, sleep 1s and wait for reconnect")
				time.Sleep(time.Second)
				continue
			}
			resp, err := subscribe.sdsStreamClient.streamSecretsClient.Recv()
			if err != nil {
				log.DefaultLogger.Warnf("[xds] [sds subscriber] get resp timeout: %v, retry after 1s", err)
				time.Sleep(time.Second)
				continue
			}
			subscribe.handleSecretResp(resp)
		}
	}
}

func (subscribe *SdsSubscriber) sendRequest(request *xdsapi.DiscoveryRequest) error {
	err := subscribe.sdsStreamClient.streamSecretsClient.Send(request)
	if err != nil {
		return err
	}
	return nil
}

func (subscribe *SdsSubscriber) handleSecretResp(response *xdsapi.DiscoveryResponse) {
	for _, res := range response.Resources {
		secret := auth.Secret{}
		secret.Unmarshal(res.GetValue())
		subscribe.provider.SetSecret(secret.Name, &secret)
	}
}

func (subscribe *SdsSubscriber) getSdsStreamClient(sdsStreamConfig *SdsStreamConfig) (*SdsStreamClient, error) {
	if subscribe.sdsStreamClient != nil {
		return subscribe.sdsStreamClient, nil
	}
	udsPath := "unix://" + sdsStreamConfig.sdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	sdsServiceClient := v2.NewSecretDiscoveryServiceClient(conn)
	sdsStreamClient := &SdsStreamClient{
		sdsStreamConfig: sdsStreamConfig,
		conn:            conn,
	}
	ctx, cancel := context.WithCancel(context.Background())
	sdsStreamClient.cancel = cancel
	streamSecretsClient, err := sdsServiceClient.StreamSecrets(ctx)
	if err != nil {
		conn.Close()
		return nil, err
	}
	sdsStreamClient.streamSecretsClient = streamSecretsClient
	return sdsStreamClient, nil
}

func (subscribe *SdsSubscriber) reconnect() {
	subscribe.sdsStreamClient.cancel()
	if subscribe.sdsStreamClient.conn != nil {
		subscribe.sdsStreamClient.conn.Close()
		subscribe.sdsStreamClient.conn = nil
	}
	sdsStreamConfig := subscribe.sdsStreamClient.sdsStreamConfig
	subscribe.sdsStreamClient = nil
	log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed")
	for {
		sdsStreamClient, err := subscribe.getSdsStreamClient(sdsStreamConfig)
		if err != nil {
			log.DefaultLogger.Warnf("[xds] [sds subscriber] stream client reconnect failed, retry after 1s")
			time.Sleep(time.Second)
			continue
		}
		subscribe.sdsStreamClient = sdsStreamClient
		log.DefaultLogger.Infof("[xds] [sds subscriber] stream client reconnected")
		break
	}
}
