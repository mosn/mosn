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

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

type SdsStreamClientImpl struct {
	conn                *grpc.ClientConn
	cancel              context.CancelFunc
	streamSecretsClient v2.SecretDiscoveryService_StreamSecretsClient
}

var _ sds.SdsStreamClient = (*SdsStreamClientImpl)(nil)

func init() {
	sds.RegisterSdsStreamClientFactory(CreateSdsStreamClient)
}

func CreateSdsStreamClient(config interface{}) (sds.SdsStreamClient, error) {
	sdsConfig, err := convertConfig(config)
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
	sdsServiceClient := v2.NewSecretDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	sdsStreamClient := &SdsStreamClientImpl{
		conn:   conn,
		cancel: cancel,
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
	request := &xdsapi.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: []string{name},
		TypeUrl:       "type.googleapis.com/envoy.api.v2.auth.Secret",
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node: &envoy_api_v2_core.Node{
			Id: istio.GetGlobalXdsInfo().ServiceNode,
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
		secret := &auth.Secret{}
		ptypes.UnmarshalAny(res, secret)
		provider.SetSecret(secret.Name, convertSecret(secret))
	}
	if callback != nil {
		callback()
	}
	// send ack response
	if err := sc.AckResponse(resp); err != nil {
		// ack resposne send failed does not returns an error
		// becasue we handle the response finished
		log.DefaultLogger.Errorf("ack response secret fail: %v", err)
	}
	return nil
}

func (sc *SdsStreamClientImpl) AckResponse(resp *xdsapi.DiscoveryResponse) error {
	secretNames := make([]string, 0)
	for _, resource := range resp.Resources {
		if resource == nil {
			continue
		}

		secret := &auth.Secret{}
		if err := ptypes.UnmarshalAny(resource, secret); err != nil {
			log.DefaultLogger.Errorf("fail to extract secret name: %v", err)
			continue
		}

		secretNames = append(secretNames, secret.GetName())
	}

	return sc.streamSecretsClient.Send(&xdsapi.DiscoveryRequest{
		VersionInfo:   resp.VersionInfo,
		ResourceNames: secretNames,
		TypeUrl:       resp.TypeUrl,
		ResponseNonce: resp.Nonce,
		ErrorDetail:   nil,
		Node: &envoy_api_v2_core.Node{
			Id: istio.GetGlobalXdsInfo().ServiceNode,
		},
	})
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

func convertConfig(config interface{}) (SdsStreamConfig, error) {
	sdsConfig := SdsStreamConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return sdsConfig, err
	}
	source := &envoy_api_v2_core.ConfigSource{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), source); err != nil {
		return sdsConfig, err
	}
	if apiConfig, ok := source.ConfigSourceSpecifier.(*envoy_api_v2_core.ConfigSource_ApiConfigSource); ok {
		if apiConfig.ApiConfigSource.GetApiType() == envoy_api_v2_core.ApiConfigSource_GRPC {
			grpcService := apiConfig.ApiConfigSource.GetGrpcServices()
			if len(grpcService) != 1 {
				log.DefaultLogger.Alertf("sds.subscribe.grpc", "[xds] [sds subscriber] only support one grpc service,but get %v", len(grpcService))
				return sdsConfig, errors.New("unsupport sds config")
			}
			grpcConfig, ok := grpcService[0].TargetSpecifier.(*envoy_api_v2_core.GrpcService_GoogleGrpc_)
			if !ok {
				return sdsConfig, errors.New("unsupport sds target specifier")
			}
			sdsConfig.sdsUdsPath = grpcConfig.GoogleGrpc.TargetUri
			sdsConfig.statPrefix = grpcConfig.GoogleGrpc.StatPrefix
		}
	}
	return sdsConfig, nil
}

func convertSecret(raw *auth.Secret) *types.SdsSecret {
	secret := &types.SdsSecret{
		Name: raw.Name,
	}
	if validateSecret, ok := raw.Type.(*auth.Secret_ValidationContext); ok {
		ds := validateSecret.ValidationContext.TrustedCa.Specifier.(*envoy_api_v2_core.DataSource_InlineBytes)
		secret.ValidationPEM = string(ds.InlineBytes)
	}
	if tlsCert, ok := raw.Type.(*auth.Secret_TlsCertificate); ok {
		certSpec, _ := tlsCert.TlsCertificate.CertificateChain.Specifier.(*envoy_api_v2_core.DataSource_InlineBytes)
		priKey, _ := tlsCert.TlsCertificate.PrivateKey.Specifier.(*envoy_api_v2_core.DataSource_InlineBytes)
		secret.CertificatePEM = string(certSpec.InlineBytes)
		secret.PrivateKeyPEM = string(priKey.InlineBytes)
	}
	return secret
}
