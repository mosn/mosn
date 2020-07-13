package sds

import (
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/juju/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsSubscriber struct {
	provider           types.SecretProvider
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

var (
	SubscriberRetryPeriod = 3 * time.Second
)

func NewSdsSubscriber(provider types.SecretProvider, sdsConfig *core.ConfigSource, serviceNode string, serviceCluster string) *SdsSubscriber {
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

func (subscribe *SdsSubscriber) Start() {
	for {
		sdsStreamConfig, err := subscribe.convertSdsConfig(subscribe.sdsConfig)
		if err != nil {
			log.DefaultLogger.Alertf("sds.subscribe.config", "[sds][subscribe] convert sds config fail %v", err)
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		streamClient, err := subscribe.getSdsStreamClient(sdsStreamConfig)
		if err != nil {
			log.DefaultLogger.Alertf("sds.subscribe.stream", "[sds][subscribe] get sds stream client fail %v", err)
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		log.DefaultLogger.Infof("[sds][subscribe] init sds stream client success")
		subscribe.sdsStreamClient = streamClient
		break
	}
	utils.GoWithRecover(func() {
		subscribe.sendRequestLoop(subscribe.sdsStreamClient)
	}, nil)
	utils.GoWithRecover(func() {
		subscribe.receiveResponseLoop(subscribe.sdsStreamClient)
	}, nil)
}

func (subscribe *SdsSubscriber) Stop() {
	close(subscribe.sendStopChannel)
	close(subscribe.receiveStopChannel)
}

func (subscribe *SdsSubscriber) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

func (subscribe *SdsSubscriber) convertSdsConfig(sdsConfig *core.ConfigSource) (*SdsStreamConfig, error) {
	sdsStreamConfig := &SdsStreamConfig{}
	if apiConfig, ok := subscribe.sdsConfig.ConfigSourceSpecifier.(*core.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == core.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Alertf("sds.subscribe.grpc", "[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return nil, errors.New("unsupport sds config")
			}
			if grpcConfig, ok := grpcService[0].TargetSpecifier.(*core.GrpcService_GoogleGrpc_); ok {
				sdsStreamConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
				sdsStreamConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
			} else {
				return nil, errors.New("unsupport sds target specifier")
			}
		}
	}
	return sdsStreamConfig, nil
}

func (subscribe *SdsSubscriber) sendRequestLoop(sdsStreamClient *SdsStreamClient) {
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
			for {
				err := subscribe.sendRequest(discoveryReq)
				if err != nil {
					log.DefaultLogger.Alertf("sds.subscribe.request", "[xds] [sds subscriber] send sds request fail , resource name = %v", name)
					time.Sleep(1 * time.Second)
					subscribe.reconnect()
					continue
				}
				break
			}
		}
	}
}

func (subscribe *SdsSubscriber) receiveResponseLoop(sdsStreamClient *SdsStreamClient) {
	for {
		select {
		case <-subscribe.receiveStopChannel:
			return
		default:
			if subscribe.sdsStreamClient == nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, sleep 1s and wait for reconnect")
				time.Sleep(time.Second)
				continue
			}
			resp, err := subscribe.sdsStreamClient.streamSecretsClient.Recv()
			if err != nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] get resp timeout: %v, retry after 1s", err)
				time.Sleep(time.Second)
				subscribe.reconnect()
				continue
			}
			subscribe.handleSecretResp(resp)
		}
	}
}

func (subscribe *SdsSubscriber) sendRequest(request *xdsapi.DiscoveryRequest) error {
	log.DefaultLogger.Debugf("send sds request resource name = %v", request.ResourceNames)
	return subscribe.sdsStreamClient.streamSecretsClient.Send(request)
}

func (subscribe *SdsSubscriber) handleSecretResp(response *xdsapi.DiscoveryResponse) {
	log.DefaultLogger.Debugf("handle secret response %v", response)
	for _, res := range response.Resources {
		secret := &auth.Secret{}
		ptypes.UnmarshalAny(res, secret)
		subscribe.provider.SetSecret(secret.Name, secret)
	}
	if sdsPostCallback != nil {
		sdsPostCallback()
	}
}

func (subscribe *SdsSubscriber) getSdsStreamClient(sdsStreamConfig *SdsStreamConfig) (*SdsStreamClient, error) {
	if subscribe.sdsStreamClient != nil {
		return subscribe.sdsStreamClient, nil
	}
	udsPath := "unix:" + sdsStreamConfig.sdsUdsPath
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
	subscribe.sdsStreamClient = nil
	log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed")
	for {
		sdsStreamConfig, err := subscribe.convertSdsConfig(subscribe.sdsConfig)
		if err != nil {
			log.DefaultLogger.Alertf("sds.subscribe.config", "[xds][sds subscriber] convert sds config fail %v", err)
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		sdsStreamClient, err := subscribe.getSdsStreamClient(sdsStreamConfig)
		if err != nil {
			log.DefaultLogger.Infof("[xds] [sds subscriber] stream client reconnect failed, retry after 1s")
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		subscribe.sdsStreamClient = sdsStreamClient
		log.DefaultLogger.Infof("[xds] [sds subscriber] stream client reconnected")
		break
	}
}
