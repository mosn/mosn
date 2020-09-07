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

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	sds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	ptypes "github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

const (
	SecretType = "type.googleapis.com/envoy.api.v2.auth.Secret"
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
	defer CloseSdsClient()
	// wait server start and stop makes reconnect
	time.Sleep(time.Second)
	srv.Stop()
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
	defer CloseSdsClient()
	sdsClient.AddUpdateCallback(config, func(name string, secret *types.SdsSecret) {})
	err := sdsClient.DeleteUpdateCallback(config)
	if err != nil {
		t.Errorf("delete update callback fail")
	}
}

func InitSdsSecertConfig(sdsUdsPath string) *auth.SdsSecretConfig {
	gRPCConfig := &core.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &core.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &core.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &core.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
	}
	config := &auth.SdsSecretConfig{
		Name: "default",
		SdsConfig: &core.ConfigSource{
			ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
				ApiConfigSource: &core.ApiConfigSource{
					ApiType: core.ApiConfigSource_GRPC,
					GrpcServices: []*core.GrpcService{
						{
							TargetSpecifier: &core.GrpcService_GoogleGrpc_{
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
	sds.SecretDiscoveryServiceServer
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

func (s *fakeSdsServer) StreamSecrets(stream sds.SecretDiscoveryService_StreamSecretsServer) error {
	log.DefaultLogger.Infof("get stream secrets")
	// wait for request
	// for test just ignore
	_, err := stream.Recv()
	if err != nil {
		log.DefaultLogger.Errorf("streamn receive error: %v", err)
		return err
	}
	resp := &xdsapi.DiscoveryResponse{
		TypeUrl:     SecretType,
		VersionInfo: "0",
		Nonce:       "0",
	}
	secret := &auth.Secret{
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

func (s *fakeSdsServer) FetchSecrets(ctx context.Context, discReq *xdsapi.DiscoveryRequest) (*xdsapi.DiscoveryResponse, error) {
	// not implement
	return nil, nil
}

func (s *fakeSdsServer) register(rpcs *grpc.Server) {
	sds.RegisterSecretDiscoveryServiceServer(rpcs, s)
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
