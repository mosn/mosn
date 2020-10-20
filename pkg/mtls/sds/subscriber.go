package sds

import (
	"sync"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/juju/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsSubscriber struct {
	provider             types.SecretProvider
	reqQueue             chan string
	sdsConfig            *envoy_config_core_v3.ConfigSource
	sdsStreamClient      *SdsStreamClient
	sdsStreamClientMutex sync.RWMutex
	sendStopChannel      chan int
	receiveStopChannel   chan int
	serviceNode          string
	serviceCluster       string
}

type SdsStreamClient struct {
	sdsStreamConfig     *SdsStreamConfig
	conn                *grpc.ClientConn
	cancel              context.CancelFunc
	streamSecretsClient envoy_service_secret_v3.SecretDiscoveryService_StreamSecretsClient
}

type SdsStreamConfig struct {
	sdsUdsPath string
	statPrefix string
}

var (
	SubscriberRetryPeriod = 3 * time.Second
)

func NewSdsSubscriber(provider types.SecretProvider, sdsConfig *envoy_config_core_v3.ConfigSource, serviceNode string, serviceCluster string) *SdsSubscriber {
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
		if err := subscribe.getSdsStreamClient(sdsStreamConfig); err != nil {
			log.DefaultLogger.Alertf("sds.subscribe.stream", "[sds][subscribe] get sds stream client fail %v", err)
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		log.DefaultLogger.Infof("[sds][subscribe] init sds stream client success")
		break
	}
	utils.GoWithRecover(func() {
		subscribe.sendRequestLoop()
	}, nil)
	utils.GoWithRecover(func() {
		subscribe.receiveResponseLoop()
	}, nil)
}

func (subscribe *SdsSubscriber) Stop() {
	close(subscribe.sendStopChannel)
	close(subscribe.receiveStopChannel)
}

func (subscribe *SdsSubscriber) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

func (subscribe *SdsSubscriber) convertSdsConfig(sdsConfig *envoy_config_core_v3.ConfigSource) (*SdsStreamConfig, error) {
	sdsStreamConfig := &SdsStreamConfig{}
	if apiConfig, ok := subscribe.sdsConfig.ConfigSourceSpecifier.(*envoy_config_core_v3.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == envoy_config_core_v3.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Alertf("sds.subscribe.grpc", "[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return nil, errors.New("unsupport sds config")
			}
			if grpcConfig, ok := grpcService[0].TargetSpecifier.(*envoy_config_core_v3.GrpcService_GoogleGrpc_); ok {
				sdsStreamConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
				sdsStreamConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
			} else {
				return nil, errors.New("unsupport sds target specifier")
			}
		}
	}
	return sdsStreamConfig, nil
}

func (subscribe *SdsSubscriber) sendRequestLoop() {
	for {
		select {
		case <-subscribe.sendStopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber] send request loop closed")
			return
		case name := <-subscribe.reqQueue:
			discoveryReq := &envoy_service_discovery_v3.DiscoveryRequest{
				ResourceNames: []string{name},
				Node: &envoy_config_core_v3.Node{
					Id: subscribe.serviceNode,
				},
			}
			for {
				err := subscribe.sendRequest(discoveryReq)
				if err != nil {
					log.DefaultLogger.Alertf("sds.subscribe.request", "[xds] [sds subscriber] send sds request fail , resource name = %v", name)
					time.Sleep(1 * time.Second)
					// subscribe.reconnect()
					continue
				}
				break
			}
		}
	}
}

func (subscribe *SdsSubscriber) receiveResponseLoop() {
	for {
		select {
		case <-subscribe.receiveStopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber]  receive response loop closed")
			return
		default:
			subscribe.sdsStreamClientMutex.RLock()
			clt := subscribe.sdsStreamClient
			subscribe.sdsStreamClientMutex.RUnlock()

			if clt == nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, sleep 1s and wait for reconnect")
				time.Sleep(time.Second)
				continue
			}
			resp, err := clt.streamSecretsClient.Recv()
			if err != nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] get resp timeout: %v, retry after 1s", err)
				time.Sleep(time.Second)
				subscribe.reconnect()
				continue
			}
			log.DefaultLogger.Infof("[xds] [sds subscriber] received a repsonse")
			subscribe.handleSecretResp(resp)
		}
	}
}

func (subscribe *SdsSubscriber) sendRequest(request *envoy_service_discovery_v3.DiscoveryRequest) error {
	log.DefaultLogger.Debugf("send sds request resource name = %v", request.ResourceNames)

	subscribe.sdsStreamClientMutex.RLock()
	clt := subscribe.sdsStreamClient
	subscribe.sdsStreamClientMutex.RUnlock()

	if clt == nil {
		return errors.New("stream client has beend closed")
	}
	return clt.streamSecretsClient.Send(request)
}

func (subscribe *SdsSubscriber) handleSecretResp(response *envoy_service_discovery_v3.DiscoveryResponse) {
	log.DefaultLogger.Debugf("handle secret response %v", response)
	for _, res := range response.Resources {
		secret := &envoy_extensions_transport_sockets_tls_v3.Secret{}
		ptypes.UnmarshalAny(res, secret)
		subscribe.provider.SetSecret(secret.Name, secret)
	}
	if sdsPostCallback != nil {
		sdsPostCallback()
	}
}

func (subscribe *SdsSubscriber) getSdsStreamClient(sdsStreamConfig *SdsStreamConfig) error {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		return nil
	}
	udsPath := "unix:" + sdsStreamConfig.sdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		return err
	}
	sdsServiceClient := envoy_service_secret_v3.NewSecretDiscoveryServiceClient(conn)
	sdsStreamClient := &SdsStreamClient{
		sdsStreamConfig: sdsStreamConfig,
		conn:            conn,
	}
	ctx, cancel := context.WithCancel(context.Background())
	sdsStreamClient.cancel = cancel
	streamSecretsClient, err := sdsServiceClient.StreamSecrets(ctx)
	if err != nil {
		conn.Close()
		return err
	}
	sdsStreamClient.streamSecretsClient = streamSecretsClient
	subscribe.sdsStreamClient = sdsStreamClient
	return nil
}

func (subscribe *SdsSubscriber) reconnect() {
	subscribe.cleanSdsStreamClient()
	for {
		sdsStreamConfig, err := subscribe.convertSdsConfig(subscribe.sdsConfig)
		if err != nil {
			log.DefaultLogger.Alertf("sds.subscribe.config", "[xds][sds subscriber] convert sds config fail %v", err)
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		if err := subscribe.getSdsStreamClient(sdsStreamConfig); err != nil {
			log.DefaultLogger.Infof("[xds] [sds subscriber] stream client reconnect failed, retry after 1s")
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		log.DefaultLogger.Infof("[xds] [sds subscriber] stream client reconnected")
		break
	}
}

func (subscribe *SdsSubscriber) cleanSdsStreamClient() {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		subscribe.sdsStreamClient.cancel()
		if subscribe.sdsStreamClient.conn != nil {
			subscribe.sdsStreamClient.conn.Close()
			subscribe.sdsStreamClient.conn = nil
		}
		subscribe.sdsStreamClient = nil
	}
	log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed")
}
