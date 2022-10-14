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
	"crypto/x509"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_filters_common_fault_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/common/fault/v3"
	envoy_extensions_filters_http_compressor_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	envoy_extensions_filters_http_fault_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	envoy_extensions_filters_http_gzip_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/gzip/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls/extensions/sni"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/upstream/cluster"
)

func TestMain(m *testing.M) {
	// init
	router.NewRouterManager()
	cm := cluster.NewClusterManagerSingleton(nil, nil, nil)
	sc := server.NewConfig(&v2.ServerConfig{
		ServerName:      "test_xds_server",
		DefaultLogPath:  "stdout",
		DefaultLogLevel: "FATAL",
	})
	server.NewServer(sc, &mockCMF{}, cm)
	os.Exit(m.Run())
}

// messageToAny converts from proto message to proto Any
func messageToAny(t *testing.T, msg proto.Message) *any.Any {
	s, err := ptypes.MarshalAny(msg)
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
		return nil
	}
	return s
}

func Test_convertEndpointsConfig(t *testing.T) {
	type args struct {
		xdsEndpoint *envoy_config_endpoint_v3.LocalityLbEndpoints
	}
	tests := []struct {
		name string
		args args
		want []v2.Host
	}{
		{
			name: "case1",
			args: args{
				xdsEndpoint: &envoy_config_endpoint_v3.LocalityLbEndpoints{
					Priority: 1,
				},
			},
			want: []v2.Host{},
		},
		{
			name: "case2",
			args: args{
				xdsEndpoint: &envoy_config_endpoint_v3.LocalityLbEndpoints{
					LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
						{
							HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
								Endpoint: &envoy_config_endpoint_v3.Endpoint{
									Address: &envoy_config_core_v3.Address{
										Address: &envoy_config_core_v3.Address_Pipe{
											Pipe: &envoy_config_core_v3.Pipe{
												Path: "./etc/proxy",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []v2.Host{
				{
					HostConfig: v2.HostConfig{
						Address: "unix:./etc/proxy",
					},
				},
			},
		},
		{
			name: "case3",
			args: args{
				xdsEndpoint: &envoy_config_endpoint_v3.LocalityLbEndpoints{
					LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
						{
							HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
								Endpoint: &envoy_config_endpoint_v3.Endpoint{
									Address: &envoy_config_core_v3.Address{
										Address: &envoy_config_core_v3.Address_SocketAddress{
											SocketAddress: &envoy_config_core_v3.SocketAddress{
												Address: "192.168.0.1",
												PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
													PortValue: 80,
												},
											},
										},
									},
								},
							},
							LoadBalancingWeight: &wrappers.UInt32Value{Value: 20},
						},
						{
							HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
								Endpoint: &envoy_config_endpoint_v3.Endpoint{
									Address: &envoy_config_core_v3.Address{
										Address: &envoy_config_core_v3.Address_SocketAddress{
											SocketAddress: &envoy_config_core_v3.SocketAddress{
												Address: "192.168.0.2",
												PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
													PortValue: 80,
												},
											},
										},
									},
								},
							},
							LoadBalancingWeight: &wrappers.UInt32Value{Value: 0},
						},
						{
							HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
								Endpoint: &envoy_config_endpoint_v3.Endpoint{
									Address: &envoy_config_core_v3.Address{
										Address: &envoy_config_core_v3.Address_SocketAddress{
											SocketAddress: &envoy_config_core_v3.SocketAddress{
												Address: "192.168.0.3",
												PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
													PortValue: 80,
												},
											},
										},
									},
								},
							},
							LoadBalancingWeight: &wrappers.UInt32Value{Value: 200},
						},
					},
				},
			},
			want: []v2.Host{
				{
					HostConfig: v2.HostConfig{
						Address: "192.168.0.1:80",
						Weight:  20,
					},
				},
				{
					HostConfig: v2.HostConfig{
						Address: "192.168.0.2:80",
						Weight:  v2.MinHostWeight,
					},
				},
				{
					HostConfig: v2.HostConfig{
						Address: "192.168.0.3:80",
						Weight:  v2.MaxHostWeight,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertEndpointsConfig(tt.args.xdsEndpoint); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertEndpointsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertListenerConfig(t *testing.T) {
	// TODO: add it
}

func generateTransportSocket(t *testing.T, msg proto.Message) *envoy_config_core_v3.TransportSocket {
	return &envoy_config_core_v3.TransportSocket{
		ConfigType: &envoy_config_core_v3.TransportSocket_TypedConfig{
			TypedConfig: messageToAny(t, msg),
		},
	}
}

func Test_convertTLS(t *testing.T) {

	t.Run("test simple tls context convert", func(t *testing.T) {
		testcases := []struct {
			description string
			config      *envoy_config_core_v3.TransportSocket
			want        v2.TLSConfig
		}{
			{
				description: "simple static tls config in listener",
				config: generateTransportSocket(t, &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
					RequireClientCertificate: NewBoolValue(true),
					CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
						TlsCertificates: []*envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
							{
								CertificateChain: &envoy_config_core_v3.DataSource{
									Specifier: &envoy_config_core_v3.DataSource_Filename{
										Filename: "cert.pem",
									},
								},
								PrivateKey: &envoy_config_core_v3.DataSource{
									Specifier: &envoy_config_core_v3.DataSource_Filename{
										Filename: "key.pem",
									},
								},
							},
						},
						ValidationContextType: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_ValidationContext{
							ValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
								TrustedCa: &envoy_config_core_v3.DataSource{
									Specifier: &envoy_config_core_v3.DataSource_Filename{
										Filename: "rootca.pem",
									},
								},
							},
						},
						TlsParams: &envoy_extensions_transport_sockets_tls_v3.TlsParameters{
							TlsMinimumProtocolVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_2,
							TlsMaximumProtocolVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3,
							CipherSuites:              []string{"ECDHE-ECDSA-AES128-GCM-SHA256", "ECDHE-RSA-AES256-GCM-SHA384"},
							EcdhCurves:                []string{"X25519", "P-256"},
						},
					},
				}),
				want: v2.TLSConfig{
					Status:            true,
					CACert:            "rootca.pem",
					CertChain:         "cert.pem",
					PrivateKey:        "key.pem",
					VerifyClient:      true,
					RequireClientCert: true,
					CipherSuites:      "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384",
					EcdhCurves:        "X25519,P-256",
					MinVersion:        "TLSv1_2",
					MaxVersion:        "TLSv1_3",
				},
			},
			{
				description: "simple sds tls config in listener",
				config: generateTransportSocket(t, &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
					CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
						TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
							{
								Name:      "default",
								SdsConfig: &envoy_config_core_v3.ConfigSource{},
							},
						},
						ValidationContextType: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_CombinedValidationContext{
							CombinedValidationContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_CombinedCertificateValidationContext{
								ValidationContextSdsSecretConfig: &envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
									Name:      "rootca",
									SdsConfig: &envoy_config_core_v3.ConfigSource{},
								},
							},
						},
					},
				}),
				want: v2.TLSConfig{
					Status: true,
					SdsConfig: &v2.SdsConfig{
						CertificateConfig: &v2.SecretConfigWrapper{
							Name:      "default",
							SdsConfig: &envoy_config_core_v3.ConfigSource{},
						},
						ValidationConfig: &v2.SecretConfigWrapper{
							Name:      "rootca",
							SdsConfig: &envoy_config_core_v3.ConfigSource{},
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			ret := convertTLS(tc.config)
			require.Equalf(t, tc.want, ret, "case %s failed", tc.description)
		}
	})

	t.Run("test downstream tls without validation", func(t *testing.T) {
		tlsContext := &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
			RequireClientCertificate: NewBoolValue(false),
			CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
				TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
					{
						Name: "kubernetes://httpbin-credential",
						SdsConfig: &envoy_config_core_v3.ConfigSource{
							ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{},
							ResourceApiVersion:    envoy_config_core_v3.ApiVersion_V3,
						},
					},
				},
			},
		}
		transport := generateTransportSocket(t, tlsContext)
		cfg := convertTLS(transport)
		require.True(t, cfg.Status)
		require.NotNil(t, cfg.SdsConfig)
		require.True(t, cfg.SdsConfig.Valid())
	})

	t.Run("test upstream tls with sni", func(t *testing.T) {
		tlsContext := &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
			Sni: "outbound_.15010_._.istiod.istio-system.svc.cluster.local",
			CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
				TlsCertificates: []*envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
					{
						CertificateChain: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_Filename{
								Filename: "cert.pem",
							},
						},
						PrivateKey: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_Filename{
								Filename: "key.pem",
							},
						},
					},
				},
				ValidationContextType: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_CombinedValidationContext{
					CombinedValidationContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_CombinedCertificateValidationContext{
						DefaultValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
							TrustedCa: &envoy_config_core_v3.DataSource{
								Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
									InlineBytes: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUMvRENDQWVTZ0F3SUJBZ0lRZHdHUVltQ1BmUkFMWTk0dFBUZ3VQVEFOQmdrcWhraUc5dzBCQVFzRkFEQVkKTVJZd0ZBWURWUVFLRXcxamJIVnpkR1Z5TG14dlkyRnNNQjRYRFRJeU1ERXhNekExTlRnek1Gb1hEVE15TURFeApNVEExTlRnek1Gb3dHREVXTUJRR0ExVUVDaE1OWTJ4MWMzUmxjaTVzYjJOaGJEQ0NBU0l3RFFZSktvWklodmNOCkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMenloTFhBREJWcmVNRVRUQ2FTaVdnb3RtNlFuTXNuYUtNeERSdU0KZVBNTmNxR2NVM281bk9aaDJadFlIZHNNOXJkUlppbFRzS0lHWnB1djYwR08zOVVYUEdBWERoOW1hUUhhb0lkUAphejdJazRaSHNTUDlITTV3d2pTemltdGFQZFIweXl3d01kMEp4dWh0NlFJTElXd0hIU0dSVmFLUWJyYjFSRWR3CjhPMWk4VUo1eWVpSnpQMm9wSG9ocmVEUGgxVzJNUlFJdXcyemJ2cUFPbkhsTStZcG5aeXBJTVNZbm1ocm1FTm0KUFBNTXBTenBjVldzeFJjOVhicFNickJDdXFEeHlhSk80NFR0MzdGTC9qYndBbThCOG8rR3FmN0Q5VmZMWWFRcQpsRUNHYzJISUxVaG5mYStuYlYvdElLTkZlV2I4YW1ua1ZHNnRUZU5rV2pvUjMwOENBd0VBQWFOQ01FQXdEZ1lEClZSMFBBUUgvQkFRREFnSUVNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGSlhldkl5NlhSTVEKRW1jLzhSQis5WG1XWXFFUk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQVJ5R1BUdXlvNStIOHQ0ZHNrMmVQRwp1QmpmZTFDcGdTaUMwWVNERzVkY0ZxNjROU1JZSko0N1QvdkxrcEw5TWw5alM4VFdpREdEUkRlbHZiYktmNVEvCnJNUkxhNlM5YVFIc3VhRk9jNW1YbG04SDBTMDdUV0NCekFYMStXb1hLQXdyY2NuSGU1OVdmV0oyYXpLb0VMcUkKN0xvOFdCS3J0OTB3VFdvL2VRVUxYT1pjaGp0TEkxR1EzOG01ajk2TDg0QjdNMnhHVTVOUmxCTzJnemFaUG8vagpsWndPNDNzNnhLWFI4Z0hyMHJNaU1tWTJjQTFoUmtNc2NqbGZKWFczbE5VcXUxYkZ0QjV4ZTNIbXZQcjVKWVBvCithTm5ZUVNHZWxHVnpUVWcvcnlmMzhoVko2T2lZYXRSbTdicjRZY25OcHh4eEdXYTZWRTdtNlZtZmxvNzREVjgKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="),
								},
							},
							MatchSubjectAltNames: []*envoy_type_matcher_v3.StringMatcher{
								{
									MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{
										Exact: "spiffe://cluster.local/ns/istio-system/sa/istiod-service-account",
									},
								},
							},
						},
					},
				},
			},
		}
		transport := generateTransportSocket(t, tlsContext)
		cfg := convertTLS(transport)
		require.True(t, cfg.Status)
		require.NotNil(t, cfg.ExtendVerify)
		require.Equal(t, sni.SniVerify, cfg.Type)
		require.Equal(t, "spiffe://cluster.local/ns/istio-system/sa/istiod-service-account", cfg.ExtendVerify[sni.ConfigKey])
		require.Equal(t, "cert.pem", cfg.CertChain)
		require.Equal(t, "key.pem", cfg.PrivateKey)
		require.Equal(t, "outbound_.15010_._.istiod.istio-system.svc.cluster.local", cfg.ServerName)
		// verify root ca is a valid string
		p := x509.NewCertPool()
		require.True(t, p.AppendCertsFromPEM([]byte(cfg.CACert)))
	})
}

func Test_convertHeaders(t *testing.T) {
	type args struct {
		xdsHeaders []*envoy_config_route_v3.HeaderMatcher
	}
	tests := []struct {
		name string
		args args
		want []v2.HeaderMatcher
	}{
		{
			name: "case1",
			args: args{
				xdsHeaders: []*envoy_config_route_v3.HeaderMatcher{
					{
						Name: "end-user",
						HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_ExactMatch{
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

func NewBoolValue(val bool) *wrappers.BoolValue {
	return &wrappers.BoolValue{
		Value: val,
	}
}

func Test_convertCidrRange(t *testing.T) {
	type args struct {
		cidr []*envoy_config_core_v3.CidrRange
	}
	tests := []struct {
		name string
		args args
		want []v2.CidrRange
	}{
		{
			name: "case1",
			args: args{
				cidr: []*envoy_config_core_v3.CidrRange{
					{
						AddressPrefix: "192.168.1.1",
						PrefixLen:     &wrappers.UInt32Value{Value: 32},
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

func Test_convertHeadersToAdd(t *testing.T) {
	type args struct {
		headerValueOption []*envoy_config_core_v3.HeaderValueOption
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
				headerValueOption: []*envoy_config_core_v3.HeaderValueOption{
					{
						Header: &envoy_config_core_v3.HeaderValue{
							Key:   "namespace",
							Value: "demo",
						},
						Append: &wrappers.BoolValue{Value: false},
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

func Test_convertRedirectAction(t *testing.T) {
	testCases := []struct {
		name     string
		in       *envoy_config_route_v3.RedirectAction
		expected *v2.RedirectAction
	}{
		{
			name: "host redirect",
			in: &envoy_config_route_v3.RedirectAction{
				HostRedirect: "example.com",
			},
			expected: &v2.RedirectAction{
				HostRedirect: "example.com",
				ResponseCode: http.StatusMovedPermanently,
			},
		},
		{
			name: "path redirect",
			in: &envoy_config_route_v3.RedirectAction{
				PathRewriteSpecifier: &envoy_config_route_v3.RedirectAction_PathRedirect{
					PathRedirect: "/foo",
				},
			},
			expected: &v2.RedirectAction{
				PathRedirect: "/foo",
				ResponseCode: http.StatusMovedPermanently,
			},
		},
		{
			name: "scheme redirect",
			in: &envoy_config_route_v3.RedirectAction{
				SchemeRewriteSpecifier: &envoy_config_route_v3.RedirectAction_SchemeRedirect{
					SchemeRedirect: "https",
				},
			},
			expected: &v2.RedirectAction{
				SchemeRedirect: "https",
				ResponseCode:   http.StatusMovedPermanently,
			},
		},
		{
			name: "set redirect code",
			in: &envoy_config_route_v3.RedirectAction{
				HostRedirect: "example.com",
				ResponseCode: envoy_config_route_v3.RedirectAction_TEMPORARY_REDIRECT,
			},
			expected: &v2.RedirectAction{
				HostRedirect: "example.com",
				ResponseCode: http.StatusTemporaryRedirect,
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			got := convertRedirectAction(tc.in)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Unexpected redirect action\nExpected: %#v\nGot: %#v\n", tc.expected, got)
			}
		})
	}
}

func Test_convertRedirectResponseCode(t *testing.T) {
	testCases := []struct {
		name     string
		in       envoy_config_route_v3.RedirectAction_RedirectResponseCode
		expected int
	}{
		{
			name:     "redirect code 301",
			in:       envoy_config_route_v3.RedirectAction_MOVED_PERMANENTLY,
			expected: http.StatusMovedPermanently,
		},
		{
			name:     "redirect code 302",
			in:       envoy_config_route_v3.RedirectAction_FOUND,
			expected: http.StatusFound,
		},
		{
			name:     "redirect code 303",
			in:       envoy_config_route_v3.RedirectAction_SEE_OTHER,
			expected: http.StatusSeeOther,
		},
		{
			name:     "redirect code 307",
			in:       envoy_config_route_v3.RedirectAction_TEMPORARY_REDIRECT,
			expected: http.StatusTemporaryRedirect,
		},
		{
			name:     "redirect code 308",
			in:       envoy_config_route_v3.RedirectAction_PERMANENT_REDIRECT,
			expected: http.StatusPermanentRedirect,
		},
		{
			name:     "default redirect code",
			in:       306,
			expected: http.StatusMovedPermanently,
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			got := convertRedirectResponseCode(tc.in)
			if got != tc.expected {
				t.Errorf("Unexpected redirect code\nExpected: %d\nGot: %d\n", tc.expected, got)
			}
		})
	}
}

func Test_convertDirectResponseAction(t *testing.T) {
	testCases := []struct {
		name     string
		in       *envoy_config_route_v3.DirectResponseAction
		expected *v2.DirectResponseAction
	}{
		{
			name: "directResponse with body",
			in: &envoy_config_route_v3.DirectResponseAction{
				Status: 200,
				Body: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineString{
						InlineString: "directResponse with body",
					},
				},
			},
			expected: &v2.DirectResponseAction{
				StatusCode: 200,
				Body:       "directResponse with body",
			},
		},
		{
			name: "directResponse no body",
			in: &envoy_config_route_v3.DirectResponseAction{
				Status: 200,
				Body: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineString{
						InlineString: "",
					},
				},
			},
			expected: &v2.DirectResponseAction{
				StatusCode: 200,
				Body:       "",
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			got := convertDirectResponseAction(tc.in)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Unexpected directResponse action\nExpected: %#v\nGot: %#v\n", tc.expected, got)
			}
		})
	}
}

// Test stream filters convert for envoy.fault
func Test_convertStreamFilter_IsitoFault(t *testing.T) {
	faultInjectConfig := &envoy_extensions_filters_http_fault_v3.HTTPFault{
		Delay: &envoy_extensions_filters_common_fault_v3.FaultDelay{
			Percentage: &envoy_type_v3.FractionalPercent{
				Numerator:   100,
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
			FaultDelaySecifier: &envoy_extensions_filters_common_fault_v3.FaultDelay_FixedDelay{FixedDelay: &duration.Duration{Seconds: 0}},
		},
		Abort: &envoy_extensions_filters_http_fault_v3.FaultAbort{
			Percentage: &envoy_type_v3.FractionalPercent{
				Numerator:   100,
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
			ErrorType: &envoy_extensions_filters_http_fault_v3.FaultAbort_HttpStatus{
				HttpStatus: 500,
			},
		},
		UpstreamCluster: "testupstream",
		Headers: []*envoy_config_route_v3.HeaderMatcher{
			{
				Name: "end-user",
				HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_ExactMatch{
					ExactMatch: "jason",
				},
				InvertMatch: false,
			},
		},
	}

	faultStruct := messageToAny(t, faultInjectConfig)
	// empty types.Struct will makes a default empty filter
	testCases := []struct {
		config   *any.Any
		expected *v2.StreamFaultInject
	}{
		{
			config: faultStruct,
			expected: &v2.StreamFaultInject{
				Delay: &v2.DelayInject{
					Delay: 0,
					DelayInjectConfig: v2.DelayInjectConfig{
						Percent: 100,
					},
				},
				Abort: &v2.AbortInject{
					Status:  500,
					Percent: 100,
				},
				Headers: []v2.HeaderMatcher{
					{
						Name:  "end-user",
						Value: "jason",
						Regex: false,
					},
				},
				UpstreamCluster: "testupstream",
			},
		},
		{
			config:   nil,
			expected: &v2.StreamFaultInject{},
		},
	}
	for i, tc := range testCases {
		convertFilter := convertStreamFilter(wellknown.Fault, tc.config)
		if convertFilter.Type != v2.FaultStream {
			t.Errorf("#%d convert to mosn stream filter not expected, want %s, got %s", i, v2.FaultStream, convertFilter.Type)
			continue
		}
		rawFault := &v2.StreamFaultInject{}
		b, _ := json.Marshal(convertFilter.Config)
		if err := json.Unmarshal(b, rawFault); err != nil {
			t.Errorf("#%d unexpected config for fault", i)
			continue
		}
		if tc.expected.Abort == nil {
			if rawFault.Abort != nil {
				t.Errorf("#%d abort check unexpected", i)
			}
		} else {
			if rawFault.Abort.Status != tc.expected.Abort.Status || rawFault.Abort.Percent != tc.expected.Abort.Percent {
				t.Errorf("#%d abort check unexpected", i)
			}
		}
		if tc.expected.Delay == nil {
			if rawFault.Delay != nil {
				t.Errorf("#%d delay check unexpected", i)
			}
		} else {
			if rawFault.Delay.Delay != tc.expected.Delay.Delay || rawFault.Delay.Percent != tc.expected.Delay.Percent {
				t.Errorf("#%d delay check unexpected", i)
			}
		}
		if rawFault.UpstreamCluster != tc.expected.UpstreamCluster || !reflect.DeepEqual(rawFault.Headers, tc.expected.Headers) {
			t.Errorf("#%d fault config is not expected, %v", i, rawFault)
		}

	}
}

func Test_convertHashPolicy(t *testing.T) {
	xdsHashPolicy := []*envoy_config_route_v3.RouteAction_HashPolicy{
		{
			PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Header_{
				Header: &envoy_config_route_v3.RouteAction_HashPolicy_Header{
					HeaderName: "header_name",
				},
			},
		},
	}

	hp := convertHashPolicy(xdsHashPolicy)
	if !assert.NotNilf(t, hp[0].Header, "hashPolicy header field should not be nil") {
		t.FailNow()
	}
	if !assert.Equalf(t, "header_name", hp[0].Header.Key,
		"hashPolicy header field should be 'header_name'") {
		t.FailNow()
	}

	xdsHashPolicy = []*envoy_config_route_v3.RouteAction_HashPolicy{
		{
			PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie_{
				Cookie: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie{
					Name: "cookie_name",
					Path: "cookie_path",
					Ttl: &duration.Duration{
						Seconds: 5,
						Nanos:   0,
					},
				},
			},
		},
	}

	hp = convertHashPolicy(xdsHashPolicy)
	if !assert.NotNilf(t, hp[0].Cookie, "hashPolicy Cookie field should not be nil") {
		t.FailNow()
	}
	if !assert.Equalf(t, "cookie_name", hp[0].Cookie.Name,
		"hashPolicy Cookie Name field should be 'cookie_name'") {
		t.FailNow()
	}
	if !assert.Equalf(t, "cookie_path", hp[0].Cookie.Path,
		"hashPolicy Cookie Path field should be 'cookie_path'") {
		t.FailNow()
	}
	if !assert.Equalf(t, 5*time.Second, hp[0].Cookie.TTL.Duration,
		"hashPolicy Cookie TTL field should be '5s'") {
		t.FailNow()
	}

	xdsHashPolicy = []*envoy_config_route_v3.RouteAction_HashPolicy{
		{
			PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_ConnectionProperties_{
				ConnectionProperties: &envoy_config_route_v3.RouteAction_HashPolicy_ConnectionProperties{
					SourceIp: true,
				},
			},
		},
	}
	hp = convertHashPolicy(xdsHashPolicy)
	if !assert.NotNilf(t, hp[0].SourceIP, "hashPolicy SourceIP field should not be nil") {
		t.FailNow()
	}
}

// Test stream filters convert for envoy.gzip
func Test_convertStreamFilter_Gzip(t *testing.T) {
	gzipConfig := &envoy_extensions_filters_http_gzip_v3.Gzip{
		CompressionLevel: envoy_extensions_filters_http_gzip_v3.Gzip_CompressionLevel_BEST,
		Compressor: &envoy_extensions_filters_http_compressor_v3.Compressor{
			ContentLength: &wrappers.UInt32Value{Value: 1024},
			ContentType:   []string{"test"},
		},
	}

	gzipStruct := messageToAny(t, gzipConfig)
	// empty types.Struct will makes a default empty filter
	testCases := []struct {
		config   *any.Any
		expected *v2.StreamGzip
	}{
		{
			config: gzipStruct,
			expected: &v2.StreamGzip{
				GzipLevel:     fasthttp.CompressBestCompression,
				ContentLength: 1024,
				ContentType:   []string{"test"},
			},
		},
	}
	for i, tc := range testCases {
		convertFilter := convertStreamFilter(wellknown.Gzip, tc.config)
		if convertFilter.Type != v2.Gzip {
			t.Errorf("#%d convert to mosn stream filter not expected, want %s, got %s", i, v2.Gzip, convertFilter.Type)
			continue
		}
		rawGzip := &v2.StreamGzip{}
		b, _ := json.Marshal(convertFilter.Config)
		if err := json.Unmarshal(b, rawGzip); err != nil {
			t.Errorf("#%d unexpected config for fault", i)
			continue
		}

		if tc.expected.GzipLevel != rawGzip.GzipLevel {
			t.Errorf("#%d GzipLevel check unexpected", i)
		}
		if tc.expected.ContentLength != rawGzip.ContentLength {
			t.Errorf("#%d ContentLength check unexpected", i)
		}
		for k, _ := range tc.expected.ContentType {
			if tc.expected.ContentType[k] != rawGzip.ContentType[k] {
				t.Errorf("#%d ContentType check unexpected", i)
			}
		}
	}
}
