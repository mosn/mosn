/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

type SdsStreamClientImpl struct {
	conn                  *grpc.ClientConn
	cancel                context.CancelFunc
	secretDiscoveryClient envoy_service_secret_v3.SecretDiscoveryServiceClient
	streamSecretsClient   envoy_service_secret_v3.SecretDiscoveryService_StreamSecretsClient
	watchedResources      map[string]struct{}
}

var _ sds.SdsStreamClient = (*SdsStreamClientImpl)(nil)

func init() {
	sds.RegisterSdsStreamClientFactory(CreateSdsStreamClient)
}

func CreateSdsStreamClient(config interface{}) (sds.SdsStreamClient, error) {
	sdsConfig, err := ConvertConfig(config)
	if err != nil {
		log.DefaultLogger.Alertf("sds.subscribe.config", "[xds][sds subscriber] convert sds config fail %v", err)
		return nil, err
	}
	udsPath := "unix:" + sdsConfig.sdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.DefaultLogger.Alertf("sds.subscribe.stream", "[sds][subscribe] dial grpc server failed %v", err)
		return nil, err
	}
	sdsServiceClient := envoy_service_secret_v3.NewSecretDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	sdsStreamClient := &SdsStreamClientImpl{
		conn:                  conn,
		cancel:                cancel,
		secretDiscoveryClient: sdsServiceClient,
		watchedResources:      make(map[string]struct{}),
	}
	streamSecretsClient, err := sdsServiceClient.StreamSecrets(ctx)
	if err != nil {
		log.DefaultLogger.Alertf("sds.subscribe.stream", "[sds][subscribe] get sds stream secret fail %v", err)
		conn.Close()
		return nil, err
	}
	sdsStreamClient.streamSecretsClient = streamSecretsClient

	return sdsStreamClient, nil

}

func (sc *SdsStreamClientImpl) Send(name string) error {
	request := &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: []string{name},
		TypeUrl:       resource_v3.SecretType,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node: &envoy_config_core_v3.Node{
			Id:       istio.GetGlobalXdsInfo().ServiceNode,
			Cluster:  istio.GetGlobalXdsInfo().ServiceCluster,
			Metadata: istio.GetGlobalXdsInfo().Metadata,
		},
	}
	log.DefaultLogger.Debugf("send sds request resource name = %v", request.ResourceNames)
	return sc.streamSecretsClient.Send(request)
}

func (sc *SdsStreamClientImpl) Recv(provider types.SecretProvider, callback func()) error {
	resp, err := sc.streamSecretsClient.Recv()
	if err != nil {
		return err
	}
	// handle response
	log.DefaultLogger.Debugf("handle secret response %v", resp)
	for _, res := range resp.Resources {
		secret := &envoy_extensions_transport_sockets_tls_v3.Secret{}
		ptypes.UnmarshalAny(res, secret)
		provider.SetSecret(secret.Name, convertSecret(secret))
		sc.watchedResources[secret.Name] = struct{}{}
	}
	if callback != nil {
		callback()
	}
	// send ack response
	sc.AckResponse(resp)
	return nil
}

// Fetch wraps a discovery request construct and will send a grpc request without grpc options.
func (sc *SdsStreamClientImpl) Fetch(ctx context.Context, name string) (*types.SdsSecret, error) {
	resp, err := sc.secretDiscoveryClient.FetchSecrets(ctx, &envoy_service_discovery_v3.DiscoveryRequest{
		ResourceNames: []string{name},
		Node: &envoy_config_core_v3.Node{
			Id: istio.GetGlobalXdsInfo().ServiceNode,
		},
	})
	if err != nil {
		return nil, err
	}
	// TODO: need a ack request ?
	if len(resp.Resources) > 1 {
		return nil, fmt.Errorf("too many resources: %d", len(resp.Resources))
	}
	res := resp.Resources[0]
	secret := &envoy_extensions_transport_sockets_tls_v3.Secret{}
	ptypes.UnmarshalAny(res, secret)
	return convertSecret(secret), nil
}

func (sc *SdsStreamClientImpl) AckResponse(resp interface{}) {
	xdsresp, ok := resp.(*envoy_service_discovery_v3.DiscoveryResponse)
	if !ok {
		return
	}
	if err := sc.ackResponse(xdsresp); err != nil {
		log.DefaultLogger.Errorf("ack response secret fail: %v", err)
	}
}

func (sc *SdsStreamClientImpl) ackResponse(resp *envoy_service_discovery_v3.DiscoveryResponse) error {
	resourcesNames := make([]string, 0, len(sc.watchedResources))
	for k, _ := range sc.watchedResources {
		resourcesNames = append(resourcesNames, k)
	}
	ackReq := &envoy_service_discovery_v3.DiscoveryRequest{
		VersionInfo:   resp.VersionInfo,
		ResourceNames: resourcesNames,
		TypeUrl:       resp.TypeUrl,
		ResponseNonce: resp.Nonce,
		ErrorDetail:   nil,
		Node: &envoy_config_core_v3.Node{
			Id:       istio.GetGlobalXdsInfo().ServiceNode,
			Cluster:  istio.GetGlobalXdsInfo().ServiceCluster,
			Metadata: istio.GetGlobalXdsInfo().Metadata,
		},
	}
	// TODO: use ack Queue to makes ack and request by sequence
	return sc.streamSecretsClient.Send(ackReq)
}

func (sc *SdsStreamClientImpl) Stop() {
	sc.cancel()
	if sc.conn != nil {
		sc.conn.Close()
		sc.conn = nil
	}
}

type SdsStreamConfig struct {
	sdsUdsPath string
	statPrefix string
}

func ConvertConfig(config interface{}) (SdsStreamConfig, error) {
	sdsConfig := SdsStreamConfig{}
	source := &envoy_config_core_v3.ConfigSource{}

	switch v := config.(type) {
	case map[string]interface{}:
		// config from json unmarshal, we should transfer it with jsonpb
		data, err := json.Marshal(config)
		if err != nil {
			return sdsConfig, err
		}
		if err := jsonpb.Unmarshal(bytes.NewReader(data), source); err != nil {
			return sdsConfig, err
		}

	case *envoy_config_core_v3.ConfigSource:
		source = v
	default:
		return sdsConfig, errors.New("invalid config type")
	}
	if apiConfig, ok := source.ConfigSourceSpecifier.(*envoy_config_core_v3.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == envoy_config_core_v3.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Alertf("sds.subscribe.grpc", "[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return sdsConfig, errors.New("unsupport sds config")
			}
			if grpcConfig, ok := grpcService[0].TargetSpecifier.(*envoy_config_core_v3.GrpcService_GoogleGrpc_); ok {
				sdsConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
				sdsConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
			} else if _, ok := grpcService[0].TargetSpecifier.(*envoy_config_core_v3.GrpcService_EnvoyGrpc_); ok {
				//FIXME: get path from cluster
				sdsConfig.sdsUdsPath = "./etc/istio/proxy/SDS"
			} else {
				return sdsConfig, errors.New("unsupport sds target specifier")
			}
		}
	}
	return sdsConfig, nil
}

func convertSecret(raw *envoy_extensions_transport_sockets_tls_v3.Secret) *types.SdsSecret {
	secret := &types.SdsSecret{
		Name: raw.Name,
	}
	if validateSecret, ok := raw.Type.(*envoy_extensions_transport_sockets_tls_v3.Secret_ValidationContext); ok {
		ds := validateSecret.ValidationContext.TrustedCa.Specifier.(*envoy_config_core_v3.DataSource_InlineBytes)
		secret.ValidationPEM = string(ds.InlineBytes)
	}
	if tlsCert, ok := raw.Type.(*envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate); ok {
		certSpec, _ := tlsCert.TlsCertificate.CertificateChain.Specifier.(*envoy_config_core_v3.DataSource_InlineBytes)
		priKey, _ := tlsCert.TlsCertificate.PrivateKey.Specifier.(*envoy_config_core_v3.DataSource_InlineBytes)
		secret.CertificatePEM = string(certSpec.InlineBytes)
		secret.PrivateKeyPEM = string(priKey.InlineBytes)
	}
	return secret
}
