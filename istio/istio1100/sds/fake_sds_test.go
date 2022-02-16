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

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/certtool"
)

type secret struct {
	cert string
	key  string
	ca   string
}

type fakeSdsServerV3 struct {
	envoy_service_secret_v3.SecretDiscoveryServiceServer
	server     *grpc.Server
	sdsUdsPath string
	started    bool
	secret     secret
}

func InitMockSdsServer(sdsUdsPath string, t *testing.T) *fakeSdsServerV3 {
	s := NewFakeSdsServerV3(sdsUdsPath, t)
	// start server
	go func() {
		err := s.Start()
		if !s.started {
			t.Fatalf("server %s failed: %v", sdsUdsPath, err)
		}
	}()
	return s
}

func NewFakeSdsServerV3(sdsUdsPath string, t *testing.T) *fakeSdsServerV3 {
	// create a certificate for response
	priv, err := certtool.GeneratePrivateKey("RSA")
	require.Nil(t, err)
	tmpl, err := certtool.CreateTemplate("mosn fake sds", false, nil)
	require.Nil(t, err)
	cert, err := certtool.SignCertificate(tmpl, priv)
	require.Nil(t, err)
	return &fakeSdsServerV3{
		sdsUdsPath: sdsUdsPath,
		secret: secret{
			cert: cert.CertPem,
			key:  cert.KeyPem,
			ca:   certtool.GetRootCA().CertPem,
		},
	}

}

func (s *fakeSdsServerV3) Start() error {
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(10240),
	}
	grpcWorkloadServer := grpc.NewServer(grpcOptions...)
	s.register(grpcWorkloadServer)
	s.server = grpcWorkloadServer
	ln, err := setUpUdsV3(s.sdsUdsPath)
	if err != nil {
		return err
	}
	s.started = true
	err = s.server.Serve(ln)
	return err
}

func (s *fakeSdsServerV3) Stop() {
	if s.started {
		s.server.Stop()
	}
}

func (s *fakeSdsServerV3) generateSecretResource(name string) (*any.Any, error) {
	var secret *envoy_extensions_transport_sockets_tls_v3.Secret

	switch name {
	case "default":
		secret = &envoy_extensions_transport_sockets_tls_v3.Secret{
			Name: "default",
			Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
				TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
					CertificateChain: &envoy_config_core_v3.DataSource{
						Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
							InlineBytes: []byte(s.secret.cert),
						},
					},
					PrivateKey: &envoy_config_core_v3.DataSource{
						Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
							InlineBytes: []byte(s.secret.key),
						},
					},
				},
			},
		}
	case "rootca":
		secret = &envoy_extensions_transport_sockets_tls_v3.Secret{
			Name: "rootca",
			Type: &envoy_extensions_transport_sockets_tls_v3.Secret_ValidationContext{
				ValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
					TrustedCa: &envoy_config_core_v3.DataSource{
						Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
							InlineBytes: []byte(s.secret.ca),
						},
					},
				},
			},
		}

	}
	return ptypes.MarshalAny(secret)
}

func (s *fakeSdsServerV3) StreamSecrets(stream envoy_service_secret_v3.SecretDiscoveryService_StreamSecretsServer) error {
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
		resource, err := s.generateSecretResource(req.ResourceNames[0])
		if err != nil {
			log.DefaultLogger.Errorf("create secret error: %v", err)
			return err
		}
		resp := &envoy_service_discovery_v3.DiscoveryResponse{
			TypeUrl:     resourcev3.SecretType,
			VersionInfo: "0",
			Nonce:       "0",
			Resources:   []*any.Any{resource},
		}
		if err := stream.Send(resp); err != nil {
			log.DefaultLogger.Errorf("send response error: %v", err)
			return err
		}
	}

	return nil
}

func (s *fakeSdsServerV3) FetchSecrets(ctx context.Context, discReq *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	resource, err := s.generateSecretResource(discReq.ResourceNames[0])
	if err != nil {
		log.DefaultLogger.Errorf("create secret error: %v", err)
		return nil, err
	}
	resp := &envoy_service_discovery_v3.DiscoveryResponse{
		TypeUrl:     resourcev3.SecretType,
		VersionInfo: "0",
		Nonce:       "0",
		Resources:   []*any.Any{resource},
	}
	return resp, nil
}

func (s *fakeSdsServerV3) register(rpcs *grpc.Server) {
	envoy_service_secret_v3.RegisterSecretDiscoveryServiceServer(rpcs, s)
}

func setUpUdsV3(udsPath string) (net.Listener, error) {
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
