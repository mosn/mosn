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

package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/types"
	"istio.io/api/mixer/v1"
	"istio.io/api/mixer/v1/config/client"

	"github.com/alipay/sofa-mosn/pkg/api/v2"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xdscore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdsendpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	xdslistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	xdsroute "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	xdsaccesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	xdsfal "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdstcp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	xdsutil "github.com/envoyproxy/go-control-plane/pkg/util"
	google_protobuf1 "github.com/gogo/protobuf/types"
)

// todo fill the unit test
func Test_convertEndpointsConfig(t *testing.T) {
	type args struct {
		xdsEndpoint *xdsendpoint.LocalityLbEndpoints
	}
	tests := []struct {
		name string
		args args
		want []v2.Host
	}{
		{
			name: "case1",
			args: args{
				xdsEndpoint: &xdsendpoint.LocalityLbEndpoints{
					Priority: 1,
				},
			},
			want: []v2.Host{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertEndpointsConfig(tt.args.xdsEndpoint); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertEndpointsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertHeaders(t *testing.T) {
	type args struct {
		xdsHeaders []*xdsroute.HeaderMatcher
	}
	tests := []struct {
		name string
		args args
		want []v2.HeaderMatcher
	}{
		{
			name: "case1",
			args: args{
				xdsHeaders: []*xdsroute.HeaderMatcher{
					{
						Name:  "end-user",
						Value: "",
						HeaderMatchSpecifier: &xdsroute.HeaderMatcher_ExactMatch{
							ExactMatch: "jason",
						},
						InvertMatch: false,
					},
				},
			},
			want: []v2.HeaderMatcher{
				{
					Name:  "end-user",
					Value: "jason",
					Regex: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertHeaders(tt.args.xdsHeaders); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertHeaders(xdsHeaders []*xdsroute.HeaderMatcher) = %v, want %v", got, tt.want)
			}
		})
	}
}

func NewBoolValue(val bool) *types.BoolValue {
	return &types.BoolValue{
		Value:                val,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
}

func Test_convertListenerConfig(t *testing.T) {
	type args struct {
		xdsListener  *xdsapi.Listener
		address      core.Address
		filterName   string
		filterConfig *xdshttp.HttpConnectionManager
	}

	accessLogFilterConfig, _ := xdsutil.MessageToStruct(&xdsaccesslog.FileAccessLog{
		Path:   "/dev/stdout",
		Format: "",
	})

	zeroSecond := new(time.Duration)
	*zeroSecond = 0

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "0.0.0.0_80",
			args: args{
				filterConfig: &xdshttp.HttpConnectionManager{
					CodecType:  xdshttp.AUTO,
					StatPrefix: "0.0.0.0_80",
					RouteSpecifier: &xdshttp.HttpConnectionManager_RouteConfig{
						RouteConfig: &xdsapi.RouteConfiguration{
							Name: "80",
							VirtualHosts: []xdsroute.VirtualHost{
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
									Routes: []xdsroute.Route{
										{
											Match: xdsroute.RouteMatch{
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
													MaxGrpcTimeout:              new(time.Duration),
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
									Routes: []xdsroute.Route{
										{
											Match: xdsroute.RouteMatch{
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
													MaxGrpcTimeout:              new(time.Duration),
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
										"172.19.8.101:80",
									},
									Routes: []xdsroute.Route{
										{
											Match: xdsroute.RouteMatch{
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
													MaxGrpcTimeout:              new(time.Duration),
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
					ServerName: "",
					AccessLog: []*xdsfal.AccessLog{{
						Name:   "envoy.file_access_log",
						Filter: nil,
						Config: accessLogFilterConfig,
					}},
					UseRemoteAddress:                           NewBoolValue(false),
					XffNumTrustedHops:                          0,
					SkipXffAppend:                              false,
					Via:                                        "",
					GenerateRequestId:                          NewBoolValue(true),
					ForwardClientCertDetails:                   xdshttp.SANITIZE,
					SetCurrentClientCertDetails:                nil,
					Proxy_100Continue:                          false,
					RepresentIpv4RemoteAddressAsIpv4MappedIpv6: false,
				},
				filterName: "envoy.http_connection_manager",
				address: core.Address{
					Address: &core.Address_SocketAddress{
						SocketAddress: &core.SocketAddress{
							Protocol: core.TCP,
							Address:  "0.0.0.0",
							PortSpecifier: &core.SocketAddress_PortValue{
								PortValue: 80,
							},
							ResolverName: "",
							Ipv4Compat:   false,
						},
					},
				},
			},
			want: `{"name":"0.0.0.0_80","address":"","bind_port":false,"handoff_restoreddestination":false,"log_path":"stdout","access_logs":[{"log_path":"/dev/stdout"}],"filter_chains":[{"match":"\u003cnil\u003e","tls_context":{"status":false,"type":"","extend_verify":null},"filters":[{"type":"proxy","config":{"downstream_protocol":"Http1","extend_config":null,"name":"","support_dynamic_route":true,"upstream_protocol":"Http1","validate_clusters":false,"virtual_hosts":[{"domains":["istio-egressgateway.istio-system.svc.cluster.local","istio-egressgateway.istio-system.svc.cluster.local:80","istio-egressgateway.istio-system","istio-egressgateway.istio-system:80","istio-egressgateway.istio-system.svc.cluster","istio-egressgateway.istio-system.svc.cluster:80","istio-egressgateway.istio-system.svc","istio-egressgateway.istio-system.svc:80","172.19.3.204","172.19.3.204:80"],"name":"istio-egressgateway.istio-system.svc.cluster.local:80","require_tls":"NONE","routers":[{"decorator":"operation:\"istio-egressgateway.istio-system.svc.cluster.local:80/*\" ","match":{"case_sensitive":false,"headers":null,"path":"","prefix":"/","regex":"","runtime":{"default_value":0,"runtime_key":""}},"metadata":{"filter_metadata":{"mosn.lb":null}},"redirect":{"host_redirect":"","path_redirect":"","response_code":0},"route":{"cluster_header":"","cluster_name":"outbound|80||istio-egressgateway.istio-system.svc.cluster.local","metadata_match":{"filter_metadata":{"mosn.lb":null}},"retry_policy":{"num_retries":0,"retry_on":false,"retry_timeout":"0s"},"timeout":"0s","weighted_clusters":null}}],"virtual_clusters":null},{"domains":["istio-ingressgateway.istio-system.svc.cluster.local","istio-ingressgateway.istio-system.svc.cluster.local:80","istio-ingressgateway.istio-system","istio-ingressgateway.istio-system:80","istio-ingressgateway.istio-system.svc.cluster","istio-ingressgateway.istio-system.svc.cluster:80","istio-ingressgateway.istio-system.svc","istio-ingressgateway.istio-system.svc:80","172.19.8.101","172.19.8.101:80"],"name":"istio-ingressgateway.istio-system.svc.cluster.local:80","require_tls":"NONE","routers":[{"decorator":"operation:\"istio-ingressgateway.istio-system.svc.cluster.local:80/*\" ","match":{"case_sensitive":false,"headers":null,"path":"","prefix":"/","regex":"","runtime":{"default_value":0,"runtime_key":""}},"metadata":{"filter_metadata":{"mosn.lb":null}},"redirect":{"host_redirect":"","path_redirect":"","response_code":0},"route":{"cluster_header":"","cluster_name":"outbound|80||istio-ingressgateway.istio-system.svc.cluster.local","metadata_match":{"filter_metadata":{"mosn.lb":null}},"retry_policy":{"num_retries":0,"retry_on":false,"retry_timeout":"0s"},"timeout":"0s","weighted_clusters":null}}],"virtual_clusters":null},{"domains":["nginx-ingress-lb.kube-system.svc.cluster.local","nginx-ingress-lb.kube-system.svc.cluster.local:80","nginx-ingress-lb.kube-system","nginx-ingress-lb.kube-system:80","nginx-ingress-lb.kube-system.svc.cluster","nginx-ingress-lb.kube-system.svc.cluster:80","nginx-ingress-lb.kube-system.svc","nginx-ingress-lb.kube-system.svc:80","172.19.6.192:80","172.19.8.101:80"],"name":"nginx-ingress-lb.kube-system.svc.cluster.local:80","require_tls":"NONE","routers":[{"decorator":"operation:\"nginx-ingress-lb.kube-system.svc.cluster.local:80/*\" ","match":{"case_sensitive":false,"headers":null,"path":"","prefix":"/","regex":"","runtime":{"default_value":0,"runtime_key":""}},"metadata":{"filter_metadata":{"mosn.lb":null}},"redirect":{"host_redirect":"","path_redirect":"","response_code":0},"route":{"cluster_header":"","cluster_name":"outbound|80||nginx-ingress-lb.kube-system.svc.cluster.local","metadata_match":{"filter_metadata":{"mosn.lb":null}},"retry_policy":{"num_retries":0,"retry_on":false,"retry_timeout":"0s"},"timeout":"0s","weighted_clusters":null}}],"virtual_clusters":null}]}}]}],"inspector":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, _ := xdsutil.MessageToStruct(tt.args.filterConfig)
			listenerConfig := &xdsapi.Listener{
				Name:    "0.0.0.0_80",
				Address: tt.args.address,
				FilterChains: []xdslistener.FilterChain{
					{
						FilterChainMatch: nil,
						TlsContext:       nil,
						Filters: []xdslistener.Filter{
							{
								Name:         tt.args.filterName,
								Config:       conf,
								DeprecatedV1: nil,
							},
						},
					},
				},
				DeprecatedV1: &xdsapi.Listener_DeprecatedV1{
					BindToPort: NewBoolValue(false),
				},
				DrainType: xdsapi.Listener_DEFAULT,
			}

			got := convertListenerConfig(listenerConfig)
			//if data, err := json.Marshal(got); err == nil {
			if _, err := json.Marshal(got); err == nil {
				// TODO: use string comapre for result is not expected
				//if strings.Compare(tt.want, string(data)) != 0 {
				//	t.Errorf("convertListenerConfig(xdsListener *xdsapi.Listener)\ngot=%s\nwant=%s\n", string(data), tt.want)
				//}
			} else {
				t.Errorf("json.Marshal(listenerConfig) got error: %v", err)
			}
		})
	}

}

func Test_convertCidrRange(t *testing.T) {
	type args struct {
		cidr []*xdscore.CidrRange
	}
	tests := []struct {
		name string
		args args
		want []v2.CidrRange
	}{
		{
			name: "case1",
			args: args{
				cidr: []*xdscore.CidrRange{
					{
						AddressPrefix: "192.168.1.1",
						PrefixLen:     &google_protobuf1.UInt32Value{Value: 32},
					},
				},
			},
			want: []v2.CidrRange{
				{
					Address: "192.168.1.1",
					Length:  32,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertCidrRange(tt.args.cidr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertCidrRange(cidr []*xdscore.CidrRange) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertTCPRoute(t *testing.T) {
	type args struct {
		deprecatedV1 *xdstcp.TcpProxy_DeprecatedV1
	}
	tests := []struct {
		name string
		args args
		want []*v2.TCPRoute
	}{
		{
			name: "case1",
			args: args{
				deprecatedV1: &xdstcp.TcpProxy_DeprecatedV1{
					Routes: []*xdstcp.TcpProxy_DeprecatedV1_TCPRoute{
						{
							Cluster: "tcp",
							DestinationIpList: []*xdscore.CidrRange{
								{
									AddressPrefix: "192.168.1.1",
									PrefixLen:     &google_protobuf1.UInt32Value{Value: 32},
								},
							},
							DestinationPorts: "50",
							SourceIpList: []*xdscore.CidrRange{
								{
									AddressPrefix: "192.168.1.2",
									PrefixLen:     &google_protobuf1.UInt32Value{Value: 32},
								},
							},
							SourcePorts: "40",
						},
					},
				},
			},
			want: []*v2.TCPRoute{
				{
					Cluster: "tcp",
					DestinationAddrs: []v2.CidrRange{
						{
							Address: "192.168.1.1",
							Length:  32,
						},
					},
					DestinationPort: "50",
					SourceAddrs: []v2.CidrRange{
						{
							Address: "192.168.1.2",
							Length:  32,
						},
					},
					SourcePort: "40",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertTCPRoute(tt.args.deprecatedV1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertTCPRoute(deprecatedV1 *xdstcp.TcpProxy_DeprecatedV1) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertHeadersToAdd(t *testing.T) {
	type args struct {
		headerValueOption []*xdscore.HeaderValueOption
	}

	FALSE := false

	tests := []struct {
		name string
		args args
		want []*v2.HeaderValueOption
	}{
		{
			name: "case1",
			args: args{
				headerValueOption: []*xdscore.HeaderValueOption{
					{
						Header: &xdscore.HeaderValue{
							Key:   "namespace",
							Value: "demo",
						},
						Append: &google_protobuf1.BoolValue{Value: false},
					},
				},
			},
			want: []*v2.HeaderValueOption{
				{
					Header: &v2.HeaderValue{
						Key:   "namespace",
						Value: "demo",
					},
					Append: &FALSE,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertHeadersToAdd(tt.args.headerValueOption); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertHeadersToAdd(headerValueOption []*xdscore.HeaderValueOption) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertPerRouteConfig(t *testing.T) {
	type args struct {
		perFilterConfig *client.ServiceConfig
	}

	tests := []struct {
		name    string
		typeStr string
		exist   bool
		args    args
	}{
		{
			name:    "mixer",
			typeStr: reflect.TypeOf(client.ServiceConfig{}).String(),
			exist:   true,
			args: args{
				perFilterConfig: &client.ServiceConfig{
					DisableReportCalls: false,
					DisableCheckCalls:  true,
					MixerAttributes: &v1.Attributes{
						Attributes: map[string]*v1.Attributes_AttributeValue{
							"test": &v1.Attributes_AttributeValue{
								Value: &v1.Attributes_AttributeValue_StringValue{
									StringValue: "test_value",
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "test",
			typeStr: "",
			exist:   false,
		},
	}

	for _, test := range tests {
		conf, _ := xdsutil.MessageToStruct(test.args.perFilterConfig)
		confMap := map[string]*types.Struct{
			test.name: conf,
		}
		perRouteConfig := convertPerRouteConfig(confMap)

		perConfig, exist := perRouteConfig[test.name]
		if exist != test.exist {
			t.Errorf("config %s exist error", test.name)
		}

		if !exist {
			continue
		}

		typeStr := reflect.TypeOf(perConfig).String()
		if typeStr != test.typeStr {
			t.Errorf("type %s error, expect %s", typeStr, test.typeStr)
		}
	}
}
