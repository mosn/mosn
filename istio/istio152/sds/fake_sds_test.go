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

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_sds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"mosn.io/mosn/pkg/log"
)

const (
	SecretType = "type.googleapis.com/envoy.api.v2.auth.Secret"
)

// SecretDiscoveryServiceServer is the server API for SecretDiscoveryService service.
//type SecretDiscoveryServiceServer interface {
//        DeltaSecrets(SecretDiscoveryService_DeltaSecretsServer) error
//        StreamSecrets(SecretDiscoveryService_StreamSecretsServer) error
//        FetchSecrets(context.Context, *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error)
//}
type fakeSdsServer struct {
	envoy_sds.SecretDiscoveryServiceServer
	server           *grpc.Server
	sdsUdsPath       string
	started          bool
	requiredMetadata map[string]string
}

func InitMockSdsServer(t *testing.T, sdsUdsPath string, requiredMeta map[string]string) *fakeSdsServer {
	s := NewFakeSdsServer(sdsUdsPath, requiredMeta)
	go func() {
		err := s.Start()
		if !s.started {
			t.Fatalf("server %s failed: %v", sdsUdsPath, err)
		}
	}()
	return s
}

func NewFakeSdsServer(sdsUdsPath string, requiredMeta map[string]string) *fakeSdsServer {
	return &fakeSdsServer{
		sdsUdsPath:       sdsUdsPath,
		requiredMetadata: requiredMeta,
	}

}

func (s *fakeSdsServer) register(rpcs *grpc.Server) {
	envoy_sds.RegisterSecretDiscoveryServiceServer(rpcs, s)
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

// Start a grpc server as sds server
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

func (s *fakeSdsServer) matchMetadata(md metadata.MD) bool {
	for k, v := range s.requiredMetadata {
		if md.Get(k)[0] != v {
			return false
		}
	}
	return true
}

func (s *fakeSdsServer) StreamSecrets(stream envoy_sds.SecretDiscoveryService_StreamSecretsServer) error {
	log.DefaultLogger.Infof("get stream secrets")
	// wait for request
	for {
		req, err := stream.Recv()
		if err != nil {
			log.DefaultLogger.Errorf("streamn receive error: %v", err)
			return err
		}
		// mock ack
		if req.VersionInfo != "" {
			log.DefaultLogger.Infof("server receive a ack request: %v", req)
			continue
		}
		if len(s.requiredMetadata) > 0 {
			if md, ok := metadata.FromIncomingContext(stream.Context()); !ok || !s.matchMetadata(md) {
				return fmt.Errorf("metadata not found, md: %v, require: %v", md, s.requiredMetadata)
			}
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
	}
	return nil
}

func (s *fakeSdsServer) FetchSecrets(ctx context.Context, discReq *xdsapi.DiscoveryRequest) (*xdsapi.DiscoveryResponse, error) {
	if len(s.requiredMetadata) > 0 {
		if md, ok := metadata.FromIncomingContext(ctx); !ok || !s.matchMetadata(md) {
			return nil, fmt.Errorf("metadata not found, md: %v, require: %v", md, s.requiredMetadata)
		}
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
		return nil, err
	}
	resp.Resources = append(resp.Resources, ms)
	return resp, nil
}
