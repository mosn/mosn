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

package v2

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/xds/conv"
)

//  Init parsed ds and clusters config for xds
func (c *XDSConfig) Init(dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources) error {
	err := c.loadClusters(staticResources)
	if err != nil {
		return err
	}
	err = c.loadADSConfig(dynamicResources)
	if err != nil {
		return err
	}
	return nil
}

func (c *XDSConfig) loadADSConfig(dynamicResources *bootstrap.Bootstrap_DynamicResources) error {
	if dynamicResources == nil || dynamicResources.AdsConfig == nil {
		log.DefaultLogger.Errorf("DynamicResources is null")
		err := errors.New("null point exception")
		return err
	}
	err := dynamicResources.AdsConfig.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("Invalid DynamicResources")
		return err
	}
	config, err := c.getAPISourceEndpoint(dynamicResources.AdsConfig)
	if err != nil {
		log.DefaultLogger.Errorf("fail to get api source endpoint")
		return err
	}
	c.ADSConfig = config
	return nil
}

func (c *XDSConfig) getAPISourceEndpoint(source *core.ApiConfigSource) (*ADSConfig, error) {
	config := &ADSConfig{}
	if source.ApiType != core.ApiConfigSource_GRPC {
		log.DefaultLogger.Errorf("unsupported api type: %v", source.ApiType)
		err := errors.New("only support GRPC api type yet")
		return nil, err
	}
	config.APIType = source.ApiType
	if source.RefreshDelay == nil || source.RefreshDelay.GetSeconds() <= 0 {
		duration := time.Duration(time.Second * 10) // default refresh delay
		config.RefreshDelay = &duration
	} else {
		duration := conv.ConvertDuration(source.RefreshDelay)
		config.RefreshDelay = &duration
	}

	config.Services = make([]*ServiceConfig, 0, len(source.GrpcServices))
	for _, service := range source.GrpcServices {
		t := service.TargetSpecifier
		if target, ok := t.(*core.GrpcService_EnvoyGrpc_); ok {
			serviceConfig := ServiceConfig{}
			if service.Timeout == nil || (service.Timeout.Seconds <= 0 && service.Timeout.Nanos <= 0) {
				duration := time.Duration(time.Second) // default connection timeout
				serviceConfig.Timeout = &duration
			} else {
				var nanos = service.Timeout.Seconds*int64(time.Second) + int64(service.Timeout.Nanos)
				duration := time.Duration(nanos)
				serviceConfig.Timeout = &duration
			}
			clusterName := target.EnvoyGrpc.ClusterName
			serviceConfig.ClusterConfig = c.Clusters[clusterName]
			if serviceConfig.ClusterConfig == nil {
				log.DefaultLogger.Errorf("cluster not found: %s", clusterName)
				return nil, fmt.Errorf("cluster not found: %s", clusterName)
			}
			config.Services = append(config.Services, &serviceConfig)
		} else if _, ok := t.(*core.GrpcService_GoogleGrpc_); ok {
			log.DefaultLogger.Warnf("GrpcService_GoogleGrpc_ not support yet")
			continue
		}
	}
	return config, nil
}

func (c *XDSConfig) loadClusters(staticResources *bootstrap.Bootstrap_StaticResources) error {
	if staticResources == nil {
		log.DefaultLogger.Errorf("StaticResources is null")
		err := errors.New("null point exception")
		return err
	}
	err := staticResources.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("Invalid StaticResources")
		return err
	}
	c.Clusters = make(map[string]*ClusterConfig)
	for _, cluster := range staticResources.Clusters {
		name := cluster.Name
		config := ClusterConfig{}
		config.TlsContext = cluster.TlsContext
		if cluster.LbPolicy != xdsapi.Cluster_RANDOM {
			log.DefaultLogger.Warnf("only random lbPoliy supported, convert to random")
		}
		config.LbPolicy = xdsapi.Cluster_RANDOM
		if cluster.ConnectTimeout.GetSeconds() <= 0 {
			duration := time.Second * 10
			config.ConnectTimeout = &duration // default connect timeout
		} else {
			duration := conv.ConvertDuration(cluster.ConnectTimeout)
			config.ConnectTimeout = &duration
		}
		config.Address = make([]string, 0, len(cluster.Hosts))
		for _, host := range cluster.Hosts {
			if address, ok := host.Address.(*core.Address_SocketAddress); ok {
				if port, ok := address.SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue); ok {
					newAddress := fmt.Sprintf("%s:%d", address.SocketAddress.Address, port.PortValue)
					config.Address = append(config.Address, newAddress)
				} else {
					log.DefaultLogger.Warnf("only PortValue supported")
					continue
				}
			} else {
				log.DefaultLogger.Warnf("only SocketAddress supported")
				continue
			}
		}
		c.Clusters[name] = &config
	}
	return nil
}

// GetEndpoint return an endpoint address by random
func (c *ClusterConfig) GetEndpoint() (string, *time.Duration) {
	if c.LbPolicy != xdsapi.Cluster_RANDOM || len(c.Address) < 1 {
		// never happen
		return "", nil
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := r.Intn(len(c.Address))

	return c.Address[idx], c.ConnectTimeout
}

// GetStreamClient return a grpc stream client that connected to ads
func (c *ADSConfig) GetStreamClient() ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient {
	c.StreamClientMutex.RLock()
	adsSc := c.StreamClient
	c.StreamClientMutex.RUnlock()

	if adsSc != nil && adsSc.Client != nil {
		return adsSc.Client
	}

	c.StreamClientMutex.Lock()
	defer c.StreamClientMutex.Unlock()

	adsSc = c.StreamClient
	if adsSc != nil && adsSc.Client != nil {
		return adsSc.Client
	}

	sc := &StreamClient{}

	if c.Services == nil {
		log.DefaultLogger.Errorf("no available ads service")
		return nil
	}
	var endpoint string
	var tlsContext *envoy_api_v2_auth.UpstreamTlsContext
	var timeout *time.Duration
	for _, service := range c.Services {
		if service.ClusterConfig == nil {
			continue
		}
		endpoint, _ = service.ClusterConfig.GetEndpoint()
		if len(endpoint) > 0 {
			timeout = service.ClusterConfig.ConnectTimeout
			tlsContext = service.ClusterConfig.TlsContext
			break
		}
	}
	if len(endpoint) == 0 {
		log.DefaultLogger.Errorf("no available ads endpoint")
		return nil
	}

	var timeoutCtx context.Context
	if timeout != nil && *timeout > 0 {
		timeoutCtx, _ = context.WithTimeout(context.Background(), *timeout)
	}

	var conn *grpc.ClientConn
	var err error

	if tlsContext == nil || !featuregate.Enabled(featuregate.XdsMtlsEnable) {
		if timeoutCtx != nil {
			conn, err = grpc.DialContext(timeoutCtx, endpoint, grpc.WithInsecure(), generateDialOption())
		} else {
			conn, err = grpc.Dial(endpoint, grpc.WithInsecure(), generateDialOption())
		}
		if err != nil {
			log.DefaultLogger.Errorf("did not connect: %v", err)
			return nil
		}
		log.DefaultLogger.Infof("mosn estab grpc connection to pilot at %v", endpoint)
		sc.Conn = conn
	} else {
		// Grpc with mTls support
		creds, err := c.getTLSCreds(tlsContext)
		if err != nil {
			log.DefaultLogger.Errorf("xds-grpc get tls creds fail: err= %v", err)
			return nil
		}
		if timeoutCtx != nil {
			conn, err = grpc.DialContext(timeoutCtx, endpoint, grpc.WithTransportCredentials(creds), generateDialOption())
		} else {
			conn, err = grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), generateDialOption())
		}
		if err != nil {
			log.DefaultLogger.Errorf("did not connect: %v", err)
			return nil
		}
		log.DefaultLogger.Infof("mosn estab grpc connection to pilot at %v", endpoint)
		sc.Conn = conn
	}
	client := ads.NewAggregatedDiscoveryServiceClient(sc.Conn)
	var ctx context.Context
	var cancel context.CancelFunc
	if timeoutCtx == nil {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithCancel(timeoutCtx)
	}
	sc.Cancel = cancel
	streamClient, err := client.StreamAggregatedResources(ctx)
	if err != nil {
		log.DefaultLogger.Infof("fail to create stream client: %v", err)
		if sc.Conn != nil {
			sc.Conn.Close()
		}
		return nil
	}
	sc.Client = streamClient
	c.StreamClient = sc
	return streamClient
}

func (c *ADSConfig) getTLSCreds(tlsContext *envoy_api_v2_auth.UpstreamTlsContext) (credentials.TransportCredentials, error) {
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

func (c *ADSConfig) getADSRefreshDelay() *time.Duration {
	return c.RefreshDelay
}

func (c *ADSConfig) closeADSStreamClient() {
	c.StreamClientMutex.Lock()
	sc := c.StreamClient
	c.StreamClient = nil
	c.StreamClientMutex.Unlock()

	if sc == nil {
		return
	}
	sc.Cancel()
	if sc.Conn != nil {
		sc.Conn.Close()
		sc.Conn = nil
	}
	sc.Client = nil
	sc = nil
}

// [xds] [ads client] get resp timeout: rpc error: code = ResourceExhausted desc = grpc: received message larger than max (5193322 vs. 4194304), retry after 1s
// https://github.com/istio/istio/blob/9686754643d0939c1f4dd0ee20443c51183f3589/pilot/pkg/bootstrap/server.go#L662
// Istio xDS DiscoveryServer not set grpc MaxSendMsgSize. If this is not set, gRPC uses the default `math.MaxInt32`.
func generateDialOption() grpc.DialOption {
	return grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(math.MaxInt32),
	)
}
