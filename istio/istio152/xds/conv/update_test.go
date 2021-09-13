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

package conv

import (
	"context"
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	xdslistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	xdsroute "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	xdshttpfault "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/fault/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdstcp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	xdsutil "github.com/envoyproxy/go-control-plane/pkg/conversion"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/duration"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
)

func messageToStruct(t *testing.T, msg proto.Message) *pstruct.Struct {
	s, err := xdsutil.MessageToStruct(msg)
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	return s
}

func Test_updateListener(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestUpdateListener error: %v", r)
		}
	}()
	cvt := NewConverter()

	// no support access log register
	//	accessLogFilterConfig := messageToStruct(t, &xdsaccesslog.FileAccessLog{
	//		Path: "/dev/stdout",
	//	})
	zeroSecond := new(duration.Duration)
	zeroSecond.Seconds = 0
	filterConfig := &xdshttp.HttpConnectionManager{
		CodecType:  xdshttp.HttpConnectionManager_AUTO,
		StatPrefix: "0.0.0.0_80",
		RouteSpecifier: &xdshttp.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoy_api_v2.RouteConfiguration{
				Name: "80",
				VirtualHosts: []*xdsroute.VirtualHost{
					{
						Name: "istio-egressgateway.istio-system.svc.cluster.local:80",
						Domains: []string{
							"istio-egressgateway.istio-system.svc.cluster.local",
							"istio-egressgateway.istio-system.svc.cluster.local:80",
							"istio-egressgateway.istio-system",
							"istio-egressgateway.istio-system:80",
							"istio-egressgateway.istio-system.svc.cluster",
							"istio-egressgateway.istio-system.svc.cluster:80",
							"istio-egressgateway.istio-system.svc",
							"istio-egressgateway.istio-system.svc:80",
							"172.19.3.204",
							"172.19.3.204:80",
						},
						Routes: []*xdsroute.Route{
							{
								Match: &xdsroute.RouteMatch{
									PathSpecifier: &xdsroute.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &xdsroute.Route_Route{
									Route: &xdsroute.RouteAction{
										ClusterSpecifier: &xdsroute.RouteAction_Cluster{
											Cluster: "outbound|80||istio-egressgateway.istio-system.svc.cluster.local",
										},
										ClusterNotFoundResponseCode: xdsroute.RouteAction_SERVICE_UNAVAILABLE,
										MetadataMatch:               nil,
										PrefixRewrite:               "",
										HostRewriteSpecifier:        nil,
										Timeout:                     zeroSecond,
										RetryPolicy:                 nil,
										RequestMirrorPolicy:         nil,
										Priority:                    core.RoutingPriority_DEFAULT,
										MaxGrpcTimeout:              new(duration.Duration),
									},
								},
								Metadata: nil,
								Decorator: &xdsroute.Decorator{
									Operation: "istio-egressgateway.istio-system.svc.cluster.local:80/*",
								},
								PerFilterConfig: nil,
							},
						},
						RequireTls: xdsroute.VirtualHost_NONE,
					},

					{
						Name: "istio-ingressgateway.istio-system.svc.cluster.local:80",
						Domains: []string{
							"istio-ingressgateway.istio-system.svc.cluster.local",
							"istio-ingressgateway.istio-system.svc.cluster.local:80",
							"istio-ingressgateway.istio-system",
							"istio-ingressgateway.istio-system:80",
							"istio-ingressgateway.istio-system.svc.cluster",
							"istio-ingressgateway.istio-system.svc.cluster:80",
							"istio-ingressgateway.istio-system.svc",
							"istio-ingressgateway.istio-system.svc:80",
							"172.19.8.101",
							"172.19.8.101:80",
						},
						Routes: []*xdsroute.Route{
							{
								Match: &xdsroute.RouteMatch{
									PathSpecifier: &xdsroute.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &xdsroute.Route_Route{
									Route: &xdsroute.RouteAction{
										ClusterSpecifier: &xdsroute.RouteAction_Cluster{
											Cluster: "outbound|80||istio-ingressgateway.istio-system.svc.cluster.local",
										},
										ClusterNotFoundResponseCode: xdsroute.RouteAction_SERVICE_UNAVAILABLE,
										PrefixRewrite:               "",
										Timeout:                     zeroSecond,
										Priority:                    core.RoutingPriority_DEFAULT,
										MaxGrpcTimeout:              new(duration.Duration),
									},
								},
								Decorator: &xdsroute.Decorator{
									Operation: "istio-ingressgateway.istio-system.svc.cluster.local:80/*",
								},
								PerFilterConfig: nil,
							},
						},
						RequireTls: xdsroute.VirtualHost_NONE,
					},
					{
						Name: "nginx-ingress-lb.kube-system.svc.cluster.local:80",
						Domains: []string{
							"nginx-ingress-lb.kube-system.svc.cluster.local",
							"nginx-ingress-lb.kube-system.svc.cluster.local:80",
							"nginx-ingress-lb.kube-system",
							"nginx-ingress-lb.kube-system:80",
							"nginx-ingress-lb.kube-system.svc.cluster",
							"nginx-ingress-lb.kube-system.svc.cluster:80",
							"nginx-ingress-lb.kube-system.svc",
							"nginx-ingress-lb.kube-system.svc:80",
							"172.19.6.192:80",
						},
						Routes: []*xdsroute.Route{
							{
								Match: &xdsroute.RouteMatch{
									PathSpecifier: &xdsroute.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &xdsroute.Route_Route{
									Route: &xdsroute.RouteAction{
										ClusterSpecifier: &xdsroute.RouteAction_Cluster{
											Cluster: "outbound|80||nginx-ingress-lb.kube-system.svc.cluster.local",
										},
										ClusterNotFoundResponseCode: xdsroute.RouteAction_SERVICE_UNAVAILABLE,
										PrefixRewrite:               "",
										Timeout:                     zeroSecond,
										Priority:                    core.RoutingPriority_DEFAULT,
										MaxGrpcTimeout:              new(duration.Duration),
									},
								},
								Decorator: &xdsroute.Decorator{
									Operation: "nginx-ingress-lb.kube-system.svc.cluster.local:80/*",
								},
								PerFilterConfig: nil,
							},
						},
						RequireTls: xdsroute.VirtualHost_NONE,
					},
				},
				ValidateClusters: NewBoolValue(false),
			},
		},
		//AccessLog: []*xdsfal.AccessLog{{
		//	Name:   "envoy.file_access_log",
		//	Filter: nil,
		//	ConfigType: &xdsfal.AccessLog_Config{
		//		Config: accessLogFilterConfig,
		//	},
		//}},
		UseRemoteAddress:            NewBoolValue(false),
		XffNumTrustedHops:           0,
		SkipXffAppend:               false,
		Via:                         "",
		GenerateRequestId:           NewBoolValue(true),
		ForwardClientCertDetails:    xdshttp.HttpConnectionManager_SANITIZE,
		SetCurrentClientCertDetails: nil,
		Proxy_100Continue:           false,
		RepresentIpv4RemoteAddressAsIpv4MappedIpv6: false,
	}
	filterName := "envoy.http_connection_manager"
	address := core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.SocketAddress_TCP,
				Address:  "0.0.0.0",
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: 80,
				},
				ResolverName: "",
				Ipv4Compat:   false,
			},
		},
	}
	//
	listenerConfig := &envoy_api_v2.Listener{
		Name:    "0.0.0.0_80",
		Address: &address,
		FilterChains: []*xdslistener.FilterChain{
			{
				Filters: []*xdslistener.Filter{
					{
						Name: filterName,
						ConfigType: &xdslistener.Filter_Config{
							Config: messageToStruct(t, filterConfig),
						},
					},
				},
			},
		},
		DeprecatedV1: &envoy_api_v2.Listener_DeprecatedV1{
			BindToPort: NewBoolValue(false),
		},
		DrainType: envoy_api_v2.Listener_DEFAULT,
	}
	cvt.ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{listenerConfig})
	// Verify
	adapter := server.GetListenerAdapterInstance()
	if ln := adapter.FindListenerByName("", "0.0.0.0_80"); ln == nil {
		t.Fatal("no listener found")
	}
	routerMng := router.GetRoutersMangerInstance()
	if rw := routerMng.GetRouterWrapperByName("80"); rw == nil {
		t.Fatal("no router found")
	}
	// Test Update Stream Filter
	streamFilterConfig := &xdshttpfault.HTTPFault{}
	filterConfig.HttpFilters = []*xdshttp.HttpFilter{
		&xdshttp.HttpFilter{
			Name: v2.FaultStream,
			ConfigType: &xdshttp.HttpFilter_Config{
				Config: messageToStruct(t, streamFilterConfig),
			},
		},
	}
	listenerConfig.FilterChains[0].Filters[0] = &xdslistener.Filter{
		Name: filterName,
		ConfigType: &xdslistener.Filter_Config{
			Config: messageToStruct(t, filterConfig),
		},
	}
	cvt.ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{listenerConfig})
	ln := adapter.FindListenerByName("", "0.0.0.0_80")
	if ln == nil {
		t.Fatal("no listener found")
	}
	lnCfg := ln.Config()
	if len(lnCfg.StreamFilters) != 1 || lnCfg.StreamFilters[0].Type != v2.FaultStream {
		t.Fatalf("listener stream filters config is : %v", lnCfg.StreamFilters)
	}
	// tcp proxy networker
	filterName = xdswellknown.TCPProxy
	tcpConfig := &xdstcp.TcpProxy{
		StatPrefix: "tcp",
		ClusterSpecifier: &xdstcp.TcpProxy_Cluster{
			Cluster: "tcpCluster",
		},
	}

	listenerConfig = &envoy_api_v2.Listener{
		Name:    "0.0.0.0_80",
		Address: &address,
		FilterChains: []*xdslistener.FilterChain{
			{
				Filters: []*xdslistener.Filter{
					{
						Name: filterName,
						ConfigType: &xdslistener.Filter_Config{
							Config: messageToStruct(t, tcpConfig),
						},
					},
				},
			},
		},
		DeprecatedV1: &envoy_api_v2.Listener_DeprecatedV1{
			BindToPort: NewBoolValue(false),
		},
		DrainType: envoy_api_v2.Listener_DEFAULT,
	}
	cvt.ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{listenerConfig})

	cvt.ConvertDeleteListeners([]*envoy_api_v2.Listener{listenerConfig})
}

func Test_updateListener_bookinfo(t *testing.T) {
	ln1 := &envoy_api_v2.Listener{
		Name: "0.0.0.0_9080",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 9080,
					},
				},
			},
		},
		UseOriginalDst: &wrappers.BoolValue{Value: false},
		DeprecatedV1: &envoy_api_v2.Listener_DeprecatedV1{
			BindToPort: &wrappers.BoolValue{Value: false},
		},
		FilterChains: []*xdslistener.FilterChain{
			&xdslistener.FilterChain{
				FilterChainMatch: nil,
				TlsContext:       &auth.DownstreamTlsContext{},
				Filters: []*xdslistener.Filter{
					&xdslistener.Filter{
						Name: "envoy.http_connection_manager",
						ConfigType: &xdslistener.Filter_Config{
							Config: messageToStruct(t, &xdshttp.HttpConnectionManager{
								RouteSpecifier: &xdshttp.HttpConnectionManager_RouteConfig{
									RouteConfig: &envoy_api_v2.RouteConfiguration{
										Name: "test_router_name",
										VirtualHosts: []*xdsroute.VirtualHost{
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{
												Routes: []*xdsroute.Route{
													&xdsroute.Route{
														Match: &xdsroute.RouteMatch{
															PathSpecifier: &xdsroute.RouteMatch_Prefix{
																Prefix: "/",
															},
														},
														Action: &xdsroute.Route_Route{
															Route: &xdsroute.RouteAction{
																ClusterSpecifier: &xdsroute.RouteAction_Cluster{
																	Cluster: "outbound|9080||reviews.default.svc.cluster.local",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							}),
						},
					},
				},
			},
		},
	}
	cvt := NewConverter()
	cvt.ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{ln1})
	// verify listener
	adapter := server.GetListenerAdapterInstance()
	ln := adapter.FindListenerByName("", "0.0.0.0_9080")
	require.NotNil(t, ln)
	listener := ln.Config()
	if listener.Name != "0.0.0.0_9080" || listener.BindToPort || len(listener.FilterChains) != 1 {
		t.Fatalf("error listener[0.0.0.0_9080] config: %v", listener)
	}

	if len(listener.FilterChains[0].Filters) != 1 {
		t.Fatalf("error listener[0.0.0.0_9080] config: %v", listener)
	}
	// verify router
	func(t *testing.T) {
		routerMng := router.GetRoutersMangerInstance()
		rw := routerMng.GetRouterWrapperByName("test_router_name")
		require.NotNil(t, rw)
		rcfg := rw.GetRoutersConfig()
		vhs := rcfg.VirtualHosts
		if len(vhs) != 4 {
			t.Fatalf("listener[0.0.0.0_9080] virtual hosts is not 3, got %d", len(vhs))
		}
		vh := vhs[3]
		routers := vh.Routers
		clusterName := routers[0].Route.ClusterName
		if clusterName != "outbound|9080||reviews.default.svc.cluster.local" {
			t.Fatalf("reviews.default.svc.cluster.local:9080 should route to [outbound|9080||reviews.default.svc.cluster.local], but got %s", clusterName)
		}

	}(t)
	// update router by update listener
	ln2 := &envoy_api_v2.Listener{
		Name: "0.0.0.0_9080",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 9080,
					},
				},
			},
		},
		UseOriginalDst: &wrappers.BoolValue{Value: false},
		DeprecatedV1: &envoy_api_v2.Listener_DeprecatedV1{
			BindToPort: &wrappers.BoolValue{Value: false},
		},
		FilterChains: []*xdslistener.FilterChain{
			{
				FilterChainMatch: nil,
				TlsContext:       &auth.DownstreamTlsContext{},
				Filters: []*xdslistener.Filter{
					{
						Name: "envoy.http_connection_manager",
						ConfigType: &xdslistener.Filter_Config{
							Config: messageToStruct(t, &xdshttp.HttpConnectionManager{
								RouteSpecifier: &xdshttp.HttpConnectionManager_RouteConfig{
									RouteConfig: &envoy_api_v2.RouteConfiguration{
										Name: "test_router_name",
										VirtualHosts: []*xdsroute.VirtualHost{
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{},
											&xdsroute.VirtualHost{
												Routes: []*xdsroute.Route{
													&xdsroute.Route{
														Match: &xdsroute.RouteMatch{
															PathSpecifier: &xdsroute.RouteMatch_Prefix{
																Prefix: "/",
															},
														},
														Action: &xdsroute.Route_Route{
															Route: &xdsroute.RouteAction{
																ClusterSpecifier: &xdsroute.RouteAction_WeightedClusters{
																	WeightedClusters: &xdsroute.WeightedCluster{
																		Clusters: []*xdsroute.WeightedCluster_ClusterWeight{
																			&xdsroute.WeightedCluster_ClusterWeight{
																				Name:   "outbound|9080|v1|reviews.default.svc.cluster.local",
																				Weight: &wrappers.UInt32Value{Value: 50},
																			},
																			&xdsroute.WeightedCluster_ClusterWeight{
																				Name:   "outbound|9080|v3|reviews.default.svc.cluster.local",
																				Weight: &wrappers.UInt32Value{Value: 50},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							}),
						},
					},
				},
			},
		},
	}
	cvt.ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{ln2})
	// verify router
	func(t *testing.T) {
		routerMng := router.GetRoutersMangerInstance()
		rw := routerMng.GetRouterWrapperByName("test_router_name")
		require.NotNil(t, rw)
		rcfg := rw.GetRoutersConfig()
		vhs := rcfg.VirtualHosts
		if len(vhs) != 4 {
			t.Fatalf("listener[0.0.0.0_9080] virtual hosts is not 3, got %d", len(vhs))
		}
		vh := vhs[3]
		router := vh.Routers[0].Route
		if router.ClusterName != "" {
			t.Fatalf("cluster_name is not omitempty: %s", router.ClusterName)
		}
		if len(router.WeightedClusters) != 2 {
			t.Fatalf("reviews.default.svc.cluster.local:9080 should route to weighted_clusters")
		}
		clusterName1 := router.WeightedClusters[0].Cluster.Name
		clusterName2 := router.WeightedClusters[1].Cluster.Name
		weight1 := router.WeightedClusters[0].Cluster.Weight
		weight2 := router.WeightedClusters[1].Cluster.Weight
		// 第二次 review，按照 v1 和 v3 版本各 50% 的权重路由
		if clusterName1 != "outbound|9080|v1|reviews.default.svc.cluster.local" || weight1 != 50 ||
			clusterName2 != "outbound|9080|v3|reviews.default.svc.cluster.local" || weight2 != 50 {
			t.Fatalf("reviews.default.svc.cluster.local:9080 should route to v1(50) & v3(50)")
		}
	}(t)

}

func Test_updateCluster(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestUpdateCluster error: %v", r)
		}
	}()
	cvt := NewConverter()

	addrsConfig := &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.SocketAddress_TCP,
				Address:  "localhost",
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: 80,
				},
			},
		},
	}

	ClusterConfig := &envoy_api_v2.Cluster{
		Name:                 "addCluser",
		ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{Type: envoy_api_v2.Cluster_EDS},
		EdsClusterConfig: &envoy_api_v2.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{},
		},
		LbPolicy:       envoy_api_v2.Cluster_ROUND_ROBIN,
		Hosts:          []*core.Address{addrsConfig},
		ConnectTimeout: &duration.Duration{Seconds: 1},
	}

	if mc := ConvertClustersConfig([]*envoy_api_v2.Cluster{ClusterConfig}); mc == nil {
		t.Error("ConvertClustersConfig failed!")
	}

	cvt.ConvertUpdateClusters([]*envoy_api_v2.Cluster{ClusterConfig})
	cvt.ConvertDeleteClusters([]*envoy_api_v2.Cluster{ClusterConfig})
}

func Test_updateCluster_bookinfo(t *testing.T) {
	c := &envoy_api_v2.Cluster{
		Name:     "outbound|9080||productpage.default.svc.cluster.local",
		LbPolicy: envoy_api_v2.Cluster_ROUND_ROBIN,
		Hosts: []*core.Address{
			&core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address: "172.16.1.171",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 9080,
						},
					},
				},
			},
		},
	}
	cvt := NewConverter()
	cvt.ConvertUpdateClusters([]*envoy_api_v2.Cluster{c})
	// verify cluster
	adpt := cluster.GetClusterMngAdapterInstance()
	snap := adpt.GetClusterSnapshot(context.Background(), "outbound|9080||productpage.default.svc.cluster.local")
	require.NotNil(t, snap)
	require.Equal(t, types.RoundRobin, snap.ClusterInfo().LbType())
	require.Equal(t, 1, snap.HostNum(nil))
	host := snap.LoadBalancer().ChooseHost(nil)
	require.NotNil(t, host)
	require.Equal(t, "172.16.1.171:9080", host.AddressString())
}

func Test_ConvertAddOrUpdateRouters(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestConvertAddOrUpdateRouters error: %v", r)
		}
	}()
	cvt := NewConverter()
	want := "addRoute"
	routeConfig := &envoy_api_v2.RouteConfiguration{
		Name: want,
		VirtualHosts: []*xdsroute.VirtualHost{
			{
				Name: "istio-egressgateway.istio-system.svc.cluster.local:80",
				Domains: []string{
					"istio-egressgateway.istio-system.svc.cluster.local",
					"istio-egressgateway.istio-system.svc:80",
					"172.19.3.204",
					"172.19.3.204:80",
				},
				Routes: []*xdsroute.Route{
					{
						Match: &xdsroute.RouteMatch{
							PathSpecifier: &xdsroute.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &xdsroute.Route_Route{
							Route: &xdsroute.RouteAction{
								ClusterSpecifier: &xdsroute.RouteAction_Cluster{
									Cluster: "outbound|80||istio-egressgateway.istio-system.svc.cluster.local",
								},
								ClusterNotFoundResponseCode: xdsroute.RouteAction_SERVICE_UNAVAILABLE,
								MetadataMatch:               nil,
								PrefixRewrite:               "",
								HostRewriteSpecifier:        nil,
								RetryPolicy:                 nil,
								RequestMirrorPolicy:         nil,
								Priority:                    core.RoutingPriority_DEFAULT,
								MaxGrpcTimeout:              new(duration.Duration),
							},
						},
						Metadata: nil,
						Decorator: &xdsroute.Decorator{
							Operation: "istio-egressgateway.istio-system.svc.cluster.local:80/*",
						},
						PerFilterConfig: nil,
					},
				},
				RequireTls: xdsroute.VirtualHost_NONE,
			},
		},
		ValidateClusters: NewBoolValue(false),
	}
	cvt.ConvertAddOrUpdateRouters([]*envoy_api_v2.RouteConfiguration{routeConfig})
	routersMngIns := router.GetRoutersMangerInstance()
	if routersMngIns == nil {
		t.Error("get routerMngIns failed!")
	}

	if rn := routersMngIns.GetRouterWrapperByName(want); rn == nil {
		t.Error("get routerMngIns failed!")
	}
}

func Test_ConvertUpdateEndpoints(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ConvertUpdateEndpoints error: %v", r)
		}
	}()
	cvt := NewConverter()
	cluster := "addCluser"
	ClusterConfig := &envoy_api_v2.Cluster{
		Name:                 cluster,
		ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{Type: envoy_api_v2.Cluster_EDS},
		EdsClusterConfig: &envoy_api_v2.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{},
		},
		LbPolicy: envoy_api_v2.Cluster_ROUND_ROBIN,
	}

	if mc := ConvertClustersConfig([]*envoy_api_v2.Cluster{ClusterConfig}); mc == nil {
		t.Error("ConvertClustersConfig failed!")
	}

	cvt.ConvertUpdateClusters([]*envoy_api_v2.Cluster{ClusterConfig})

	CLAConfig := &envoy_api_v2.ClusterLoadAssignment{
		ClusterName: cluster,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  "127.0.0.1",
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: 80,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}

	if err := cvt.ConvertUpdateEndpoints([]*envoy_api_v2.ClusterLoadAssignment{CLAConfig}); err != nil {
		t.Errorf("ConvertUpdateEndpoints failed: %v", err)
	}

}
