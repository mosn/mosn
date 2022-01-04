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

package xds

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"mosn.io/mosn/istio/istio152/xds/conv"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
)

type AdsConfig struct {
	APIType      core.ApiConfigSource_ApiType
	Services     []*ServiceConfig
	Clusters     map[string]*ClusterConfig
	refreshDelay *time.Duration
	xdsInfo      istio.XdsInfo
	converter    conv.Converter
}

var _ istio.XdsStreamConfig = (*AdsConfig)(nil)

func (ads *AdsConfig) CreateXdsStreamClient() (istio.XdsStreamClient, error) {
	return NewAdsStreamClient(ads)
}

const defaultRefreshDelay = time.Second * 10

func (ads *AdsConfig) RefreshDelay() time.Duration {
	if ads.refreshDelay == nil {
		return defaultRefreshDelay
	}
	return *ads.refreshDelay
}

// InitAdsRequest creates a cds request
func (ads *AdsConfig) InitAdsRequest() interface{} {
	return CreateCdsRequest(ads)
}

func (ads *AdsConfig) Node() *core.Node {
	return &core.Node{
		Id:       ads.xdsInfo.ServiceNode,
		Cluster:  ads.xdsInfo.ServiceCluster,
		Metadata: ads.xdsInfo.Metadata,
	}
}

func (ads *AdsConfig) loadADSConfig(dynamicResources *bootstrap.Bootstrap_DynamicResources) error {
	if dynamicResources == nil || dynamicResources.AdsConfig == nil {
		log.DefaultLogger.Errorf("DynamicResources is null")
		return errors.New("null point exception")
	}
	if err := dynamicResources.AdsConfig.Validate(); err != nil {
		log.DefaultLogger.Errorf("Invalid DynamicResources")
		return err
	}
	return ads.getAPISourceEndpoint(dynamicResources.AdsConfig)
}

func (ads *AdsConfig) getAPISourceEndpoint(source *core.ApiConfigSource) error {
	if source.ApiType != core.ApiConfigSource_GRPC {
		log.DefaultLogger.Errorf("unsupported api type: %v", source.ApiType)
		return errors.New("only support GRPC api type yet")
	}
	ads.APIType = source.ApiType
	if source.RefreshDelay == nil || source.RefreshDelay.GetSeconds() <= 0 {
		duration := defaultRefreshDelay
		ads.refreshDelay = &duration
	} else {
		duration := conv.ConvertDuration(source.RefreshDelay)
		ads.refreshDelay = &duration
	}
	ads.Services = make([]*ServiceConfig, 0, len(source.GrpcServices))
	for _, service := range source.GrpcServices {
		t := service.TargetSpecifier
		target, ok := t.(*core.GrpcService_EnvoyGrpc_)
		if !ok {
			continue
		}
		serviceConfig := ServiceConfig{}
		if service.Timeout == nil || (service.Timeout.Seconds <= 0 && service.Timeout.Nanos <= 0) {
			duration := time.Duration(time.Second) // default connection timeout
			serviceConfig.Timeout = &duration
		} else {
			nanos := service.Timeout.Seconds*int64(time.Second) + int64(service.Timeout.Nanos)
			duration := time.Duration(nanos)
			serviceConfig.Timeout = &duration
		}
		clusterName := target.EnvoyGrpc.ClusterName
		serviceConfig.ClusterConfig = ads.Clusters[clusterName]
		if serviceConfig.ClusterConfig == nil {
			log.DefaultLogger.Errorf("cluster not found: %s", clusterName)
			return fmt.Errorf("cluster not found: %s", clusterName)
		}
		ads.Services = append(ads.Services, &serviceConfig)
	}
	return nil
}

func (ads *AdsConfig) loadClusters(staticResources *bootstrap.Bootstrap_StaticResources) error {
	if staticResources == nil {
		log.DefaultLogger.Errorf("StaticResources is null")
		return errors.New("null point exception")
	}
	if err := staticResources.Validate(); err != nil {
		log.DefaultLogger.Errorf("Invalid StaticResources")
		return err
	}
	ads.Clusters = make(map[string]*ClusterConfig)
	for _, cluster := range staticResources.Clusters {
		name := cluster.Name
		config := ClusterConfig{}
		config.TlsContext = cluster.TlsContext
		if cluster.LbPolicy != envoy_api_v2.Cluster_RANDOM {
			log.DefaultLogger.Warnf("only random lbPolicy supported, convert to random")
		}
		config.LbPolicy = envoy_api_v2.Cluster_RANDOM
		if cluster.ConnectTimeout.GetSeconds() <= 0 {
			duration := time.Second * 10
			config.ConnectTimeout = &duration // default connect timeout
		} else {
			duration := conv.ConvertDuration(cluster.ConnectTimeout)
			config.ConnectTimeout = &duration
		}
		config.Address = make([]string, 0, len(cluster.Hosts))
		for _, host := range cluster.Hosts {
			address, ok := host.Address.(*core.Address_SocketAddress)
			if !ok {
				log.DefaultLogger.Warnf("only SocketAddress supported")
				continue
			}
			port, ok := address.SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue)
			if !ok {
				log.DefaultLogger.Warnf("only PortValue supported")
				continue
			}
			newAddress := fmt.Sprintf("%s:%d", address.SocketAddress.Address, port.PortValue)
			config.Address = append(config.Address, newAddress)
		}
		ads.Clusters[name] = &config
	}
	return nil
}

func (ads *AdsConfig) getTLSCreds(tlsContext *envoy_api_v2_auth.UpstreamTlsContext) (credentials.TransportCredentials, error) {
	if tlsContext.CommonTlsContext.GetValidationContext() == nil ||
		tlsContext.CommonTlsContext.GetValidationContext().GetTrustedCa() == nil {
		return nil, errors.New("can't find trusted ca ")
	}
	rootCAPath := tlsContext.CommonTlsContext.GetValidationContext().GetTrustedCa().GetFilename()
	if len(tlsContext.CommonTlsContext.GetTlsCertificates()) <= 0 {
		return nil, errors.New("can't find client certificates")
	}
	if tlsContext.CommonTlsContext.GetTlsCertificates()[0].GetCertificateChain() == nil ||
		tlsContext.CommonTlsContext.GetTlsCertificates()[0].GetPrivateKey() == nil {
		return nil, errors.New("can't read client certificates fail")
	}
	certChainPath := tlsContext.CommonTlsContext.GetTlsCertificates()[0].GetCertificateChain().GetFilename()
	privateKeyPath := tlsContext.CommonTlsContext.GetTlsCertificates()[0].GetPrivateKey().GetFilename()
	log.DefaultLogger.Infof("mosn start with tls context,root ca certificate path = %v\n cert chain path = %v\n private key path = %v\n",
		rootCAPath, certChainPath, privateKeyPath)
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(rootCAPath)
	if err != nil {
		return nil, err
	}
	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		return nil, errors.New("failed to append certs")
	}
	certificate, err := tls.LoadX509KeyPair(
		certChainPath,
		privateKeyPath,
	)
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "",
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})
	return creds, nil
}

// [xds] [ads client] get resp timeout: rpc error: code = ResourceExhausted desc = grpc: received message larger than max (5193322 vs. 4194304), retry after 1s
// https://github.com/istio/istio/blob/9686754643d0939c1f4dd0ee20443c51183f3589/pilot/pkg/bootstrap/server.go#L662
// Istio xDS DiscoveryServer not set grpc MaxSendMsgSize. If this is not set, gRPC uses the default `math.MaxInt32`.
func generateDialOption() grpc.DialOption {
	return grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(math.MaxInt32),
	)
}

// ServiceConfig for grpc service
type ServiceConfig struct {
	Timeout       *time.Duration
	ClusterConfig *ClusterConfig
}

// ClusterConfig contains an cluster info from static resources
type ClusterConfig struct {
	LbPolicy       envoy_api_v2.Cluster_LbPolicy
	Address        []string
	ConnectTimeout *time.Duration
	TlsContext     *envoy_api_v2_auth.UpstreamTlsContext
}

func (c *ClusterConfig) GetEndpoint() (string, *time.Duration) {
	if c.LbPolicy != envoy_api_v2.Cluster_RANDOM || len(c.Address) < 1 {
		// never happen
		return "", nil
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := r.Intn(len(c.Address))

	return c.Address[idx], c.ConnectTimeout
}
