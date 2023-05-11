package conv

import (
	"testing"

	udpa_type_v1 "github.com/cncf/udpa/go/udpa/type/v1"
	envoy_config_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_access_loggers_file_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	envoy_extensions_filters_http_fault_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	envoy_extensions_filters_listener_original_dst_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoy_extensions_filters_network_tcp_proxy_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/filter/listener/originaldst"
)

func TestConvertListener_VirtualInbound(t *testing.T) {
	virtualInboundListenerPb := &envoy_config_listener_v3.Listener{
		Name: "virtualInbound",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: 15006,
					},
				},
			},
		},
		ListenerFilters: []*envoy_config_listener_v3.ListenerFilter{
			{
				Name: "envoy.filters.listener.original_dst",
				ConfigType: &envoy_config_listener_v3.ListenerFilter_TypedConfig{
					TypedConfig: messageToAny(t, &envoy_extensions_filters_listener_original_dst_v3.OriginalDst{}),
				},
			},
		},
		FilterChains: []*envoy_config_listener_v3.FilterChain{
			// TODO: not support yet
			{
				Name: "virtualInbound-blackhole",
				FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
					DestinationPort: &wrappers.UInt32Value{
						Value: 15006,
					},
				},
				Filters: []*envoy_config_listener_v3.Filter{
					{
						Name: "envoy.filters.network.tcp_proxy",
						ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
							TypedConfig: messageToAny(t, &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
								StatPrefix: "BlackHoleCluster",
								ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
									Cluster: "BlackHoleCluster",
								},
							}),
						},
					},
				},
			},
			{
				Name: "0.0.0.0_9080",
				FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
					DestinationPort: &wrappers.UInt32Value{
						Value: 9080,
					},
				},
				Filters: []*envoy_config_listener_v3.Filter{
					{
						Name: "envoy.filters.network.http_connection_manager",
						ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
							TypedConfig: messageToAny(t, &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
								StatPrefix: "inbound_0.0.0.0_9080",
								RouteSpecifier: &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_RouteConfig{
									RouteConfig: &envoy_config_route_v3.RouteConfiguration{
										Name: "inbound|9080||",
										VirtualHosts: []*envoy_config_route_v3.VirtualHost{
											{
												Name:    "inbound|http|9080",
												Domains: []string{"*"},
												Routes: []*envoy_config_route_v3.Route{
													{
														Name: "default",
														Match: &envoy_config_route_v3.RouteMatch{
															PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{
																Prefix: "/",
															},
														},
														Action: &envoy_config_route_v3.Route_Route{
															Route: &envoy_config_route_v3.RouteAction{
																ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
																	Cluster: "inbound|9080||",
																},
																Timeout: &duration.Duration{
																	Seconds: 0,
																},
															},
														},
													},
												},
											},
										},
									},
								},
								HttpFilters: []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
									{
										Name: "envoy.filters.http.fault",
										ConfigType: &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
											TypedConfig: messageToAny(t, &envoy_extensions_filters_http_fault_v3.HTTPFault{}),
										},
									},
									{
										Name: "istio.stats",
										ConfigType: &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
											TypedConfig: messageToAny(t, &udpa_type_v1.TypedStruct{}), // TODO
										},
									},
								},
							}),
						},
					},
				},
			},
			// TODO: not support yet
			{
				Name: "virtualInbound",
				FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
					PrefixRanges: []*envoy_config_core_v3.CidrRange{
						{
							AddressPrefix: "0.0.0.0",
							PrefixLen: &wrappers.UInt32Value{
								Value: 0,
							},
						},
					},
				},
				Filters: []*envoy_config_listener_v3.Filter{
					{
						Name: "envoy.filters.network.tcp_proxy",
						ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
							TypedConfig: messageToAny(t, &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
								StatPrefix: "InboundPassthroughClusterIpv4",
								ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
									Cluster: "InboundPassthroughClusterIpv4",
								},
							}),
						},
					},
				},
			},
		},
	}
	listeners := ConvertListenerConfig(virtualInboundListenerPb, nil)
	require.Len(t, listeners, 2)
	// virtual inbound listener
	require.Equal(t, "0.0.0.0:15006", listeners[0].Addr.String())
	require.Equal(t, "virtualInbound", listeners[0].Name)
	require.Len(t, listeners[0].ListenerFilters, 1)
	require.True(t, listeners[0].BindToPort)
	b, err := json.Marshal(listeners[0].ListenerFilters[0].Config)
	require.Nil(t, err)
	cfg := originaldst.OriginalDstConfig{}
	err = json.Unmarshal(b, &cfg)
	require.Nil(t, err)
	require.True(t, cfg.FallbackToLocal)
	// bind port false listener
	require.Equal(t, "inbound|9080||", listeners[1].Name)
	require.False(t, listeners[1].BindToPort)
	require.Equal(t, "127.0.0.1:9080", listeners[1].Addr.String())
}

func TestConvertListener_UseOriginalDst(t *testing.T) {
	virtualInboundListenerPb := &envoy_config_listener_v3.Listener{
		Name: "virtualInbound",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: 15006,
					},
				},
			},
		},
		ListenerFilters: []*envoy_config_listener_v3.ListenerFilter{
			{
				Name: "envoy.filters.listener.original_dst",
				ConfigType: &envoy_config_listener_v3.ListenerFilter_TypedConfig{
					TypedConfig: messageToAny(t, &envoy_extensions_filters_listener_original_dst_v3.OriginalDst{}),
				},
			},
		},
		TrafficDirection: envoy_config_core_v3.TrafficDirection_INBOUND,
		FilterChains:     []*envoy_config_listener_v3.FilterChain{},
	}

	listeners := ConvertListenerConfig(virtualInboundListenerPb, nil)
	require.Len(t, listeners, 1)
	require.Len(t, listeners[0].ListenerFilters, 1)
	require.True(t, listeners[0].BindToPort)
	b, err := json.Marshal(listeners[0].ListenerFilters[0].Config)
	require.Nil(t, err)
	cfg := originaldst.OriginalDstConfig{}
	err = json.Unmarshal(b, &cfg)
	require.Nil(t, err)
	require.True(t, cfg.FallbackToLocal)
	require.Equal(t, "tcp_proxy", listeners[0].FilterChains[0].Filters[0].Type)
	require.Equal(t, INGRESS_CLUSTER, listeners[0].FilterChains[0].Filters[0].Config["cluster"])

	virtualOutboundListenerPb := &envoy_config_listener_v3.Listener{
		Name: "virtualOutbound",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: 15006,
					},
				},
			},
		},
		ListenerFilters: []*envoy_config_listener_v3.ListenerFilter{
			{
				Name: "envoy.filters.listener.original_dst",
				ConfigType: &envoy_config_listener_v3.ListenerFilter_TypedConfig{
					TypedConfig: messageToAny(t, &envoy_extensions_filters_listener_original_dst_v3.OriginalDst{}),
				},
			},
		},
		TrafficDirection: envoy_config_core_v3.TrafficDirection_OUTBOUND,
		FilterChains:     []*envoy_config_listener_v3.FilterChain{},
	}

	listeners = ConvertListenerConfig(virtualOutboundListenerPb, nil)
	b, err = json.Marshal(listeners[0].ListenerFilters[0].Config)
	require.Nil(t, err)
	cfg = originaldst.OriginalDstConfig{}
	err = json.Unmarshal(b, &cfg)
	require.Nil(t, err)
	require.False(t, cfg.FallbackToLocal)
	require.Equal(t, "tcp_proxy", listeners[0].FilterChains[0].Filters[0].Type)
	require.Equal(t, EGRESS_CLUSTER, listeners[0].FilterChains[0].Filters[0].Config["cluster"])
}

func TestConvertListener_AccessLogs(t *testing.T) {
	virtualListener := &envoy_config_listener_v3.Listener{
		Name: "Listener",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: 15006,
					},
				},
			},
		},
		ListenerFilters: []*envoy_config_listener_v3.ListenerFilter{
			{
				Name: "envoy.filters.listener.original_dst",
				ConfigType: &envoy_config_listener_v3.ListenerFilter_TypedConfig{
					TypedConfig: messageToAny(t, &envoy_extensions_filters_listener_original_dst_v3.OriginalDst{}),
				},
			},
		},
		FilterChains: []*envoy_config_listener_v3.FilterChain{
			// TODO: not support yet
			{
				Name: "virtualInbound-blackhole",
				FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
					DestinationPort: &wrappers.UInt32Value{
						Value: 15006,
					},
				},
				Filters: []*envoy_config_listener_v3.Filter{
					{
						Name: "envoy.filters.network.tcp_proxy",
						ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
							TypedConfig: messageToAny(t, &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
								StatPrefix: "BlackHoleCluster",
								ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
									Cluster: "BlackHoleCluster",
								},
								AccessLog: []*envoy_config_accesslog_v3.AccessLog{
									{
										Name: wellknown.FileAccessLog,
										ConfigType: &envoy_config_accesslog_v3.AccessLog_TypedConfig{
											TypedConfig: messageToAny(t, &envoy_extensions_access_loggers_file_v3.FileAccessLog{
												Path: "/tcp_proxy/log_path",
											}),
										},
									},
								},
							}),
						},
					},
				},
			},
			{
				Name: "0.0.0.0_9080",
				FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
					DestinationPort: &wrappers.UInt32Value{
						Value: 9080,
					},
				},
				Filters: []*envoy_config_listener_v3.Filter{
					{
						Name: "envoy.filters.network.http_connection_manager",
						ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
							TypedConfig: messageToAny(t, &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
								StatPrefix: "inbound_0.0.0.0_9080",
								RouteSpecifier: &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_RouteConfig{
									RouteConfig: &envoy_config_route_v3.RouteConfiguration{
										Name: "inbound|9080||",
									},
								},
								AccessLog: []*envoy_config_accesslog_v3.AccessLog{
									{
										Name: wellknown.FileAccessLog,
										ConfigType: &envoy_config_accesslog_v3.AccessLog_TypedConfig{
											TypedConfig: messageToAny(t, &envoy_extensions_access_loggers_file_v3.FileAccessLog{
												Path: "/http_connection_manager/log_path",
											}),
										},
									},
								},
							}),
						},
					},
				},
			},
		},
		AccessLog: []*envoy_config_accesslog_v3.AccessLog{
			{
				Name: wellknown.FileAccessLog,
				ConfigType: &envoy_config_accesslog_v3.AccessLog_TypedConfig{
					TypedConfig: messageToAny(t, &envoy_extensions_access_loggers_file_v3.FileAccessLog{
						Path: "/Listener/log_path",
					}),
				},
			},
		},
	}
	listeners := ConvertListenerConfig(virtualListener, nil)

	require.Len(t, listeners, 2)
	require.Len(t, listeners[0].AccessLogs, 1)
	require.Equal(t, "/Listener/log_path", listeners[0].AccessLogs[0].Path)
	require.Len(t, listeners[1].AccessLogs, 1)
	require.Equal(t, "/http_connection_manager/log_path", listeners[1].AccessLogs[0].Path)
}
