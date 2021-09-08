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
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	clusterv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_api_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

func Test_Client(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestxDSClient error: %v \n %s", r, string(debug.Stack()))
		}
	}()

	xdsConfig := XDSConfig{}
	clusterName := "xds-cluster"
	xdsAddr := "127.0.0.1"
	xdsPort := 15010
	dynamicResources := &envoy_api_bootstrap.Bootstrap_DynamicResources{
		LdsConfig: configSource(clusterName),
		CdsConfig: configSource(clusterName),
		AdsConfig: configApiSource(clusterName),
	}
	staticResources := &envoy_api_bootstrap.Bootstrap_StaticResources{
		Clusters: []*api.Cluster{{
			Name:                 clusterName,
			ConnectTimeout:       &duration.Duration{Seconds: 5},
			ClusterDiscoveryType: &api.Cluster_Type{Type: api.Cluster_STRICT_DNS},
			LbPolicy:             api.Cluster_ROUND_ROBIN,
			LoadAssignment: &api.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints: endpoints(
					socketAddress(xdsAddr, xdsPort),
				),
			},
			UpstreamConnectionOptions: &api.UpstreamConnectionOptions{
				TcpKeepalive: &envoy_api_v2_core.TcpKeepalive{
					KeepaliveProbes:   &wrappers.UInt32Value{Value: 3},
					KeepaliveTime:     &wrappers.UInt32Value{Value: 60},
					KeepaliveInterval: &wrappers.UInt32Value{Value: 6},
				},
			},
			CircuitBreakers: &clusterv2.CircuitBreakers{
				Thresholds: []*clusterv2.CircuitBreakers_Thresholds{{
					Priority:           envoy_api_v2_core.RoutingPriority_HIGH,
					MaxConnections:     &wrappers.UInt32Value{Value: 10000},
					MaxPendingRequests: &wrappers.UInt32Value{Value: 30000},
					MaxRequests:        &wrappers.UInt32Value{Value: 300000},
					MaxRetries:         &wrappers.UInt32Value{Value: 10},
				}, {
					Priority:           envoy_api_v2_core.RoutingPriority_DEFAULT,
					MaxConnections:     &wrappers.UInt32Value{Value: 30000},
					MaxPendingRequests: &wrappers.UInt32Value{Value: 30000},
					MaxRequests:        &wrappers.UInt32Value{Value: 300000},
					MaxRetries:         &wrappers.UInt32Value{Value: 300},
				}},
			},
		},
		}}

	err := xdsConfig.Init(dynamicResources, staticResources)
	if err != nil {
		t.Errorf("xDS init failed: %v", err)
	}

	adsClient := &ADSClient{
		AdsConfig:         xdsConfig.ADSConfig,
		StreamClientMutex: sync.RWMutex{},
		StreamClient:      nil,
		SendControlChan:   make(chan int),
		RecvControlChan:   make(chan int),
		StopChan:          make(chan int),
	}
	adsClient.Start()
	go adsClient.Stop()
}

// configSource returns a *envoy_api_v2_core.ConfigSource for cluster.
func configSource(cluster string) *envoy_api_v2_core.ConfigSource {
	return &envoy_api_v2_core.ConfigSource{
		ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &envoy_api_v2_core.ApiConfigSource{
				ApiType: envoy_api_v2_core.ApiConfigSource_GRPC,
				GrpcServices: []*envoy_api_v2_core.GrpcService{{
					TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
							ClusterName: cluster,
						},
					},
				}},
			},
		},
	}
}

// configSource returns a *envoy_api_v2_core.ApiConfigSource for cluster.
func configApiSource(cluster string) *envoy_api_v2_core.ApiConfigSource {
	return &envoy_api_v2_core.ApiConfigSource{
		ApiType: envoy_api_v2_core.ApiConfigSource_GRPC,
		GrpcServices: []*envoy_api_v2_core.GrpcService{{
			TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
				EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
					ClusterName: cluster,
				},
			},
		}},
	}

}

// socketAddress creates a new TCP envoy_api_v2_core.Address.
func socketAddress(address string, port int) *envoy_api_v2_core.Address {
	return &envoy_api_v2_core.Address{
		Address: &envoy_api_v2_core.Address_SocketAddress{
			SocketAddress: &envoy_api_v2_core.SocketAddress{
				Protocol: envoy_api_v2_core.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
					PortValue: uint32(port),
				},
			},
		},
	}
}

// endpoints returns a slice of LocalityLbEndpoints.
// The slice contains one entry, with one LbEndpoint per
// *envoy_api_v2_core.Address supplied.
func endpoints(addrs ...*envoy_api_v2_core.Address) []*envoy_api_v2_endpoint.LocalityLbEndpoints {
	lbendpoints := make([]*envoy_api_v2_endpoint.LbEndpoint, 0, len(addrs))
	for _, addr := range addrs {
		lbendpoints = append(lbendpoints, &envoy_api_v2_endpoint.LbEndpoint{
			HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
				Endpoint: &envoy_api_v2_endpoint.Endpoint{
					Address: addr,
				},
			},
		})
	}
	return []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
		LbEndpoints: lbendpoints,
	}}
}

func Test_Concurrent_reconnect(t *testing.T) {
	log.DefaultLogger.SetLogLevel(log.WARN)
	defer log.DefaultLogger.SetLogLevel(log.INFO)

	r := rand.Intn(100)
	xdsPort := 15555 + r
	portStr := strconv.Itoa(xdsPort)

	// 监听本地端口
	lis, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		t.Errorf("监听端口失败: %s", err)
		return
	}

	// 创建gRPC服务器
	s := grpc.NewServer()
	// 注册服务

	go func() {
		err = s.Serve(lis)
		if err != nil {
			t.Errorf("开启服务失败: %s", err)
			return
		}
	}()

	num := 10

	wg := sync.WaitGroup{}

	xdsConfig := XDSConfig{}
	clusterName := "xds-cluster"
	xdsAddr := "127.0.0.1"

	dynamicResources := &envoy_api_bootstrap.Bootstrap_DynamicResources{
		LdsConfig: configSource(clusterName),
		CdsConfig: configSource(clusterName),
		AdsConfig: configApiSource(clusterName),
	}
	staticResources := &envoy_api_bootstrap.Bootstrap_StaticResources{
		Clusters: []*api.Cluster{{
			Name:                 clusterName,
			ConnectTimeout:       &duration.Duration{Seconds: 5},
			ClusterDiscoveryType: &api.Cluster_Type{Type: api.Cluster_STRICT_DNS},
			LbPolicy:             api.Cluster_ROUND_ROBIN,
			LoadAssignment: &api.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints: endpoints(
					socketAddress(xdsAddr, xdsPort),
				),
			},
		},
		}}

	err = xdsConfig.Init(dynamicResources, staticResources)
	if err != nil {
		t.Errorf("xDS init failed: %v", err)
	}

	adsClient := &ADSClient{
		AdsConfig:         xdsConfig.ADSConfig,
		StreamClientMutex: sync.RWMutex{},
		StreamClient:      nil,
		SendControlChan:   make(chan int),
		RecvControlChan:   make(chan int),
		StopChan:          make(chan int),
	}

	adsClient.AdsConfig.Services[0].ClusterConfig.Address = []string{xdsAddr + ":" + portStr}

	adsClient.StreamClient = adsClient.AdsConfig.GetStreamClient()

	for i := 0; i < num; i++ {
		wg.Add(1)
		utils.GoWithRecover(func() {
			for i := 0; i < 1000; i++ {
				adsClient.Reconnect()
			}
			wg.Done()
		}, func(r interface{}) {
			t.Errorf("adsClient.Reconnect panic")
			wg.Done()
		})
	}
	wg.Wait()

	time.Sleep(time.Second)
	cmdStr := fmt.Sprintf("netstat -an | grep %s | grep ESTABLISHED | wc -l", portStr)
	t.Logf(cmdStr)
	c := exec.Command("bash", "-c", cmdStr)
	output, err := c.CombinedOutput()
	numStr := strings.Trim(strings.Trim(string(output), " "), "\n")
	t.Logf("numStr %s", numStr)
	numConn, err := strconv.Atoi(numStr)
	// client<->server 2 conn
	if numConn != 2 {
		t.Errorf("Reconnect conn error numConn=%s %+v", numStr, err)
	}
	s.Stop()
}
