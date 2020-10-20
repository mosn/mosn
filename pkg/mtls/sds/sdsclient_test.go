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
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	ptypes "github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func Test_AddUpdateCallback(t *testing.T) {
	// init prepare
	sdsUdsPath := "/tmp/sds2"
	SubscriberRetryPeriod = 500 * time.Millisecond
	defer func() {
		SubscriberRetryPeriod = 3 * time.Second
	}()
	callback := 0
	SetSdsPostCallback(func() {
		callback = 1
	})
	// mock sds server
	srv := InitMockSdsServer(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfig(sdsUdsPath)
	sdsClient := NewSdsClientSingleton(config)
	if sdsClient == nil {
		t.Errorf("get sds client fail")
	}

	// Do not call CloseSdsClient(), because the 'SdsClient' is a global single object.
	//defer CloseSdsClient()

	// wait server start and stop makes reconnect
	time.Sleep(time.Second)
	srv.Stop()
	// wait server connection stop
	time.Sleep(time.Second)
	// send request
	updatedChan := make(chan int, 1) // do not block the update channel
	log.DefaultLogger.Infof(" add update callback")
	sdsClient.AddUpdateCallback(config, func(name string, secret *types.SdsSecret) {
		if name != "default" {
			t.Errorf("name should same with config.name")
		}
		log.DefaultLogger.Infof("update callback is called")
		updatedChan <- 1
	})
	// sdsClient.SetSecret(config.Name, &envoy_extensions_transport_sockets_tls_v3.Secret{})
	time.Sleep(time.Second)
	go func() {
		err := srv.Start()
		if !srv.started {
			t.Fatalf("%s start error: %v", sdsUdsPath, err)
		}
	}()
	select {
	case <-updatedChan:
		if callback != 1 {
			t.Fatalf("sds post callback unexpected")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("callback reponse timeout")
	}
}
func Test_DeleteUpdateCallback(t *testing.T) {
	sdsUdsPath := "/tmp/sds3"
	// mock sds server
	srv := InitMockSdsServer(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfig(sdsUdsPath)
	config.Name = "delete"
	sdsClient := NewSdsClientSingleton(config)
	if sdsClient == nil {
		t.Errorf("get sds client fail")
	}

	// Do not call CloseSdsClient(), because the 'SdsClient' is a global single object.
	//defer CloseSdsClient()

	sdsClient.AddUpdateCallback(config, func(name string, secret *types.SdsSecret) {})
	err := sdsClient.DeleteUpdateCallback(config)
	if err != nil {
		t.Errorf("delete update callback fail")
	}
}

func InitSdsSecertConfig(sdsUdsPath string) *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig {
	gRPCConfig := &envoy_config_core_v3.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
	}
	config := &envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
		Name: "default",
		SdsConfig: &envoy_config_core_v3.ConfigSource{
			ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_ApiConfigSource{
				ApiConfigSource: &envoy_config_core_v3.ApiConfigSource{
					ApiType: envoy_config_core_v3.ApiConfigSource_GRPC,
					GrpcServices: []*envoy_config_core_v3.GrpcService{
						{
							TargetSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_{
								GoogleGrpc: gRPCConfig,
							},
						},
					},
				},
			},
		},
	}
	return config
}

func InitMockSdsServer(sdsUdsPath string, t *testing.T) *fakeSdsServer {
	s := NewFakeSdsServer(sdsUdsPath)
	go func() {
		err := s.Start()
		if !s.started {
			t.Fatalf("server %s failed: %v", sdsUdsPath, err)
		}
	}()
	return s
}

func NewFakeSdsServer(sdsUdsPath string) *fakeSdsServer {
	return &fakeSdsServer{
		sdsUdsPath: sdsUdsPath,
	}

}

// SecretDiscoveryServiceServer is the server API for SecretDiscoveryService service.
//type SecretDiscoveryServiceServer interface {
//        DeltaSecrets(SecretDiscoveryService_DeltaSecretsServer) error
//        StreamSecrets(SecretDiscoveryService_StreamSecretsServer) error
//        FetchSecrets(context.Context, *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error)
//}
type fakeSdsServer struct {
	envoy_service_secret_v3.SecretDiscoveryServiceServer
	server     *grpc.Server
	sdsUdsPath string
	started    bool
}

func (s *fakeSdsServer) Start() error {
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(10240),
	}
	grpcWorkloadServer := grpc.NewServer(grpcOptions...)
	s.register(grpcWorkloadServer)
	s.server = grpcWorkloadServer
	ln, err := setUpUds(s.sdsUdsPath)
	if err != nil {
		return err
	}
	s.started = true
	err = s.server.Serve(ln)
	return err
}

func (s *fakeSdsServer) Stop() {
	if s.started {
		s.server.Stop()
	}
}

func (s *fakeSdsServer) StreamSecrets(stream envoy_service_secret_v3.SecretDiscoveryService_StreamSecretsServer) error {
	log.DefaultLogger.Infof("get stream secrets")
	// wait for request
	// for test just ignore
	_, err := stream.Recv()
	if err != nil {
		log.DefaultLogger.Errorf("streamn receive error: %v", err)
		return err
	}
	resp := &envoy_service_discovery_v3.DiscoveryResponse{
		TypeUrl:     resourcev3.SecretType,
		VersionInfo: "0",
		Nonce:       "0",
	}
	secret := &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: "default",
	}
	ms, err := ptypes.MarshalAny(secret)
	if err != nil {
		log.DefaultLogger.Errorf("marshal secret error: %v", err)
		return err
	}
	resp.Resources = append(resp.Resources, ms)
	if err := stream.Send(resp); err != nil {
		log.DefaultLogger.Errorf("send response error: %v", err)
		return err
	}
	// keep alive for 3 second for client connection
	time.Sleep(3 * time.Second)
	return nil
}

func (s *fakeSdsServer) FetchSecrets(ctx context.Context, discReq *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	// not implement
	return nil, nil
}

func (s *fakeSdsServer) register(rpcs *grpc.Server) {
	envoy_service_secret_v3.RegisterSecretDiscoveryServiceServer(rpcs, s)
}

func setUpUds(udsPath string) (net.Listener, error) {
	// Remove unix socket before use.
	if err := os.Remove(udsPath); err != nil && !os.IsNotExist(err) {
		// Anything other than "file not found" is an error.
		return nil, fmt.Errorf("failed to remove unix://%s", udsPath)
	}

	var err error
	udsListener, err := net.Listen("unix", udsPath)
	if err != nil {
		return nil, err
	}

	// Update Sds UDS file permission so that istio-proxy has permission to access it.
	if _, err := os.Stat(udsPath); err != nil {
		return nil, fmt.Errorf("sds uds file %q doesn't exist", udsPath)
	}
	if err := os.Chmod(udsPath, 0666); err != nil {
		return nil, fmt.Errorf("failed to update %q permission", udsPath)
	}

	return udsListener, nil
}

//////
///// Deprecated with xDS2
/////

func Test_AddUpdateCallbackDeprecated(t *testing.T) {
	// init prepare
	sdsUdsPath := "/tmp/sds4"
	SubscriberRetryPeriod = 500 * time.Millisecond
	defer func() {
		SubscriberRetryPeriod = 3 * time.Second
	}()
	callback := 0
	SetSdsPostCallback(func() {
		callback = 1
	})
	// mock sds server
	srv := InitMockSdsServerDeprecated(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfigDeprecated(sdsUdsPath)
	sdsClient := NewSdsClientSingletonDeprecated(config)
	if sdsClient == nil {
		t.Errorf("get sds client fail")
	}

	// Do not call CloseSdsClient(), because the 'SdsClient' is a global single object.
	//defer CloseSdsClient()

	// wait server start and stop makes reconnect
	time.Sleep(time.Second)
	srv.Stop()
	// wait server connection stop
	time.Sleep(time.Second)
	// send request
	updatedChan := make(chan int, 1) // do not block the update channel
	log.DefaultLogger.Infof(" add update callback")
	sdsClient.AddUpdateCallbackDeprecated(config, func(name string, secret *types.SdsSecret) {
		if name != "default" {
			t.Errorf("name should same with config.name")
		}
		log.DefaultLogger.Infof("update callback is called")
		updatedChan <- 1
	})
	time.Sleep(time.Second)
	go func() {
		err := srv.Start()
		if !srv.started {
			t.Fatalf("%s start error: %v", sdsUdsPath, err)
		}
	}()
	// sdsClient.SetSecretDeprecated(config.Name, &envoy_api_v2_auth.Secret{})
	select {
	case <-updatedChan:
		if callback != 1 {
			t.Fatalf("sds post callback unexpected")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("callback reponse timeout")
	}
}

func Test_DeleteUpdateCallbackDeprecated(t *testing.T) {
	sdsUdsPath := "/tmp/sds5"
	// mock sds server
	srv := InitMockSdsServerDeprecated(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfigDeprecated(sdsUdsPath)
	config.Name = "delete"
	sdsClient := NewSdsClientSingletonDeprecated(config)
	if sdsClient == nil {
		t.Errorf("get sds client fail")
	}

	// Do not call CloseSdsClient(), because the 'SdsClient' is a global single object.
	//defer CloseSdsClient()

	sdsClient.AddUpdateCallbackDeprecated(config, func(name string, secret *types.SdsSecret) {})
	err := sdsClient.DeleteUpdateCallbackDeprecated(config)
	if err != nil {
		t.Errorf("delete update callback fail")
	}
}

func InitSdsSecertConfigDeprecated(sdsUdsPath string) *envoy_api_v2_auth.SdsSecretConfig {
	gRPCConfig := &envoy_api_v2_core.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &envoy_api_v2_core.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &envoy_api_v2_core.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &envoy_api_v2_core.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
	}
	config := &envoy_api_v2_auth.SdsSecretConfig{
		Name: "default",
		SdsConfig: &envoy_api_v2_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_ApiConfigSource{
				ApiConfigSource: &envoy_api_v2_core.ApiConfigSource{
					ApiType: envoy_api_v2_core.ApiConfigSource_GRPC,
					GrpcServices: []*envoy_api_v2_core.GrpcService{
						{
							TargetSpecifier: &envoy_api_v2_core.GrpcService_GoogleGrpc_{
								GoogleGrpc: gRPCConfig,
							},
						},
					},
				},
			},
		},
	}
	return config
}

func InitMockSdsServerDeprecated(sdsUdsPath string, t *testing.T) *fakeSdsServerDeprecated {
	s := NewFakeSdsServerDeprecated(sdsUdsPath)
	go func() {
		err := s.Start()
		if !s.started {
			t.Fatalf("server %s failed: %v", sdsUdsPath, err)
		}
	}()
	return s
}

func NewFakeSdsServerDeprecated(sdsUdsPath string) *fakeSdsServerDeprecated {
	return &fakeSdsServerDeprecated{
		sdsUdsPath: sdsUdsPath,
	}

}

// SecretDiscoveryServiceServer is the server API for SecretDiscoveryService service.
//type SecretDiscoveryServiceServer interface {
//        DeltaSecrets(SecretDiscoveryService_DeltaSecretsServer) error
//        StreamSecrets(SecretDiscoveryService_StreamSecretsServer) error
//        FetchSecrets(context.Context, *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error)
//}
type fakeSdsServerDeprecated struct {
	envoy_service_discovery_v2.SecretDiscoveryServiceServer
	server     *grpc.Server
	sdsUdsPath string
	started    bool
}

func (s *fakeSdsServerDeprecated) Start() error {
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(10240),
	}
	grpcWorkloadServer := grpc.NewServer(grpcOptions...)
	s.register(grpcWorkloadServer)
	s.server = grpcWorkloadServer
	ln, err := setUpUdsDeprecated(s.sdsUdsPath)
	if err != nil {
		return err
	}
	s.started = true
	err = s.server.Serve(ln)
	return err
}

func (s *fakeSdsServerDeprecated) Stop() {
	if s.started {
		s.server.Stop()
	}
}

func (s *fakeSdsServerDeprecated) StreamSecrets(stream envoy_service_discovery_v2.SecretDiscoveryService_StreamSecretsServer) error {
	log.DefaultLogger.Infof("get stream secrets")
	// wait for request
	// for test just ignore
	_, err := stream.Recv()
	if err != nil {
		log.DefaultLogger.Errorf("streamn receive error: %v", err)
		return err
	}
	resp := &envoy_api_v2.DiscoveryResponse{
		TypeUrl:     resourcev2.SecretType,
		VersionInfo: "0",
		Nonce:       "0",
	}
	secret := &envoy_api_v2_auth.Secret{
		Name: "default",
	}
	ms, err := ptypes.MarshalAny(secret)
	if err != nil {
		log.DefaultLogger.Errorf("marshal secret error: %v", err)
		return err
	}
	resp.Resources = append(resp.Resources, ms)
	if err := stream.Send(resp); err != nil {
		log.DefaultLogger.Errorf("send response error: %v", err)
		return err
	}
	// keep alive for 3 second for client connection
	time.Sleep(3 * time.Second)
	return nil
}

func (s *fakeSdsServerDeprecated) FetchSecrets(ctx context.Context, discReq *envoy_api_v2.DiscoveryRequest) (*envoy_api_v2.DiscoveryResponse, error) {
	// not implement
	return nil, nil
}

func (s *fakeSdsServerDeprecated) register(rpcs *grpc.Server) {
	envoy_service_discovery_v2.RegisterSecretDiscoveryServiceServer(rpcs, s)
}

func setUpUdsDeprecated(udsPath string) (net.Listener, error) {
	// Remove unix socket before use.
	if err := os.Remove(udsPath); err != nil && !os.IsNotExist(err) {
		// Anything other than "file not found" is an error.
		return nil, fmt.Errorf("failed to remove unix://%s", udsPath)
	}

	var err error
	udsListener, err := net.Listen("unix", udsPath)
	if err != nil {
		return nil, err
	}

	// Update Sds UDS file permission so that istio-proxy has permission to access it.
	if _, err := os.Stat(udsPath); err != nil {
		return nil, fmt.Errorf("sds uds file %q doesn't exist", udsPath)
	}
	if err := os.Chmod(udsPath, 0666); err != nil {
		return nil, fmt.Errorf("failed to update %q permission", udsPath)
	}

	return udsListener, nil
}
