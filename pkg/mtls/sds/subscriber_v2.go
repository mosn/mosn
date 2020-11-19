package sds

import (
	"sync"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/juju/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

//////
///// Deprecated with xDS v2
/////

type SdsSubscriberV2 struct {
	provider             types.SecretProviderV2
	reqQueue             chan string
	sdsConfig            *envoy_api_v2_core.ConfigSource
	sdsStreamClient      *SdsStreamClientV2
	sdsStreamClientMutex sync.RWMutex
	sendStopChannel      chan int
	receiveStopChannel   chan int
	serviceNode          string
	serviceCluster       string
}

type SdsStreamClientV2 struct {
	sdsStreamConfig     *SdsStreamConfigV2
	conn                *grpc.ClientConn
	cancel              context.CancelFunc
	streamSecretsClient envoy_service_discovery_v2.SecretDiscoveryService_StreamSecretsClient
}

type SdsStreamConfigV2 struct {
	sdsUdsPath string
	statPrefix string
}

func NewSdsSubscriberV2(provider types.SecretProviderV2, sdsConfig *envoy_api_v2_core.ConfigSource, serviceNode string, serviceCluster string) *SdsSubscriberV2 {
	return &SdsSubscriberV2{
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

func (subscribe *SdsSubscriberV2) Start() {
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

func (subscribe *SdsSubscriberV2) Stop() {
	close(subscribe.sendStopChannel)
	close(subscribe.receiveStopChannel)
}

func (subscribe *SdsSubscriberV2) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

func (subscribe *SdsSubscriberV2) convertSdsConfig(sdsConfig *envoy_api_v2_core.ConfigSource) (*SdsStreamConfigV2, error) {
	sdsStreamConfig := &SdsStreamConfigV2{}
	if apiConfig, ok := subscribe.sdsConfig.ConfigSourceSpecifier.(*envoy_api_v2_core.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == envoy_api_v2_core.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Alertf("sds.subscribe.grpc", "[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return nil, errors.New("unsupport sds config")
			}
			if grpcConfig, ok := grpcService[0].TargetSpecifier.(*envoy_api_v2_core.GrpcService_GoogleGrpc_); ok {
				sdsStreamConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
				sdsStreamConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
			} else {
				return nil, errors.New("unsupport sds target specifier")
			}
		}
	}
	return sdsStreamConfig, nil
}

func (subscribe *SdsSubscriberV2) sendRequestLoop() {
	for {
		select {
		case <-subscribe.sendStopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber] send request loop closed")
			return
		case name := <-subscribe.reqQueue:
			discoveryReq := &envoy_api_v2.DiscoveryRequest{
				ResourceNames: []string{name},
				Node: &envoy_api_v2_core.Node{
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

func (subscribe *SdsSubscriberV2) receiveResponseLoop() {
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

func (subscribe *SdsSubscriberV2) sendRequest(request *envoy_api_v2.DiscoveryRequest) error {
	log.DefaultLogger.Debugf("send sds request resource name = %v", request.ResourceNames)

	subscribe.sdsStreamClientMutex.RLock()
	clt := subscribe.sdsStreamClient
	subscribe.sdsStreamClientMutex.RUnlock()

	if clt == nil {
		return errors.New("stream client has beend closed")
	}
	return clt.streamSecretsClient.Send(request)
}

func (subscribe *SdsSubscriberV2) handleSecretResp(response *envoy_api_v2.DiscoveryResponse) {
	log.DefaultLogger.Debugf("handle secret response %v", response)
	for _, res := range response.Resources {
		secret := &envoy_api_v2_auth.Secret{}
		ptypes.UnmarshalAny(res, secret)
		subscribe.provider.SetSecret(secret.Name, secret)
	}
	if sdsPostCallback != nil {
		sdsPostCallback()
	}
}

func (subscribe *SdsSubscriberV2) getSdsStreamClient(sdsStreamConfig *SdsStreamConfigV2) error {
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
	sdsServiceClient := envoy_service_discovery_v2.NewSecretDiscoveryServiceClient(conn)
	sdsStreamClient := &SdsStreamClientV2{
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

func (subscribe *SdsSubscriberV2) reconnect() {
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

func (subscribe *SdsSubscriberV2) cleanSdsStreamClient() {
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
