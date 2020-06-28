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
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdslistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	xdsroute "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	xdshttpfault "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/fault/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdsutil "github.com/envoyproxy/go-control-plane/pkg/conversion"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/duration"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
)

func messageToStruct(t *testing.T, msg proto.Message) *pstruct.Struct {
	s, err := xdsutil.MessageToStruct(msg)
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	return s
}

func Test_updateListener(t *testing.T) {
	InitStats()

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
	ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{listenerConfig})
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
	ConvertAddOrUpdateListeners([]*envoy_api_v2.Listener{listenerConfig})
	ln := adapter.FindListenerByName("", "0.0.0.0_80")
	if ln == nil {
		t.Fatal("no listener found")
	}
	lnCfg := ln.Config()
	if len(lnCfg.StreamFilters) != 1 || lnCfg.StreamFilters[0].Type != v2.FaultStream {
		t.Fatalf("listener stream filters config is : %v", lnCfg.StreamFilters)
	}

}
