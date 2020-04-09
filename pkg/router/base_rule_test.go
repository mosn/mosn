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

package router

import (
	"math/rand"
	goHttp "net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func TestNilMetadataMatchCriteria(t *testing.T) {
	defaultRule := &RouteRuleImplBase{
		defaultCluster: &weightedClusterEntry{},
	}
	if defaultRule.MetadataMatchCriteria("") != nil {
		t.Errorf("nil criteria is not nil, got: %v", defaultRule.MetadataMatchCriteria(""))
	}
	weightRule := &RouteRuleImplBase{
		defaultCluster: &weightedClusterEntry{},
		weightedClusters: map[string]weightedClusterEntry{
			"test": weightedClusterEntry{},
		},
	}
	if weightRule.MetadataMatchCriteria("test") != nil {
		t.Errorf("nil criteria is not nil, got: %v", weightRule.MetadataMatchCriteria("test"))
	}
	if weightRule.MetadataMatchCriteria("") != nil {
		t.Errorf("nil criteria is not nil, got: %v", weightRule.MetadataMatchCriteria(""))
	}
}

func TestWeightedClusterSelect(t *testing.T) {
	routerMock1 := &v2.Router{}
	routerMock1.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: "defaultCluster",
			WeightedClusters: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w1",
							Weight: 90,
						},
						MetadataMatch: api.Metadata{
							"version": "v1",
						},
					},
				},
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w2",
							Weight: 10,
						},
						MetadataMatch: api.Metadata{
							"version": "v2",
						},
					},
				},
			},
		},
	}
	routerMock2 := &v2.Router{}
	routerMock2.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: "defaultCluster",
			WeightedClusters: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w1",
							Weight: 50,
						},
						MetadataMatch: api.Metadata{
							"version": "v1",
						},
					},
				},
				{
					Cluster: v2.ClusterWeight{
						ClusterWeightConfig: v2.ClusterWeightConfig{
							Name:   "w2",
							Weight: 50,
						},
						MetadataMatch: api.Metadata{
							"version": "v2",
						},
					},
				},
			},
		},
	}

	type testCase struct {
		routerCase []*v2.Router
		ratio      []uint
	}

	testCases := testCase{
		routerCase: []*v2.Router{routerMock1, routerMock2},
		ratio:      []uint{9, 1},
	}
	for index, routecase := range testCases.routerCase {
		routeRuleImplBase, _ := NewRouteRuleImplBase(nil, routecase)
		var dcCount, w1Count, w2Count uint

		totalTimes := rand.Int31n(10000)
		var i int32
		for i = 0; i < totalTimes; i++ {
			clusterName := routeRuleImplBase.ClusterName()
			switch clusterName {
			case "defaultCluster":
				dcCount++
			case "w1":
				w1Count++
			case "w2":
				w2Count++
			}
		}

		if dcCount != 0 || w1Count/w2Count < testCases.ratio[index]-1 {
			t.Errorf("cluster select wanted defalut cluster = 0, w1/w2 = 9, but got ,defalut = %d"+
				"w1/w2  = %d", dcCount, w1Count/w2Count)

		}
		t.Log("defalut = ", dcCount, "w1 = ", w1Count, "w2 =", w2Count)
	}
}

func Test_RouteRuleImplBase_matchRoute_matchMethod(t *testing.T) {
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{
				{
					Name:  "method",
					Value: "POST",
				},
			}},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: "test",
				},
			},
		},
	}

	routeRuleBase, err := NewRouteRuleImplBase(nil, route)
	if !assert.NoErrorf(t, err, "new route rule impl failed, err should be nil, get %+v", err) {
		t.FailNow()
	}

	headers := http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	headers.Set(protocol.MosnHeaderMethod, "POST")
	match := routeRuleBase.matchRoute(headers, 1)
	if !assert.Truef(t, match, "match http method failed, result should be true, get %+v", match) {
		t.FailNow()
	}

	http2Request := &goHttp.Request{
		Method: "POST",
		Header: goHttp.Header{},
	}
	headerHttp2 := http2.NewReqHeader(http2Request)
	headerHttp2.Set(protocol.MosnHeaderMethod, http2Request.Method)
	match = routeRuleBase.matchRoute(headerHttp2, 1)
	if !assert.Truef(t, match, "match http2 method failed, result should be true, get %+v", match) {
		t.FailNow()
	}
}

func Test_RouteRuleImplBase_finalizePathHeader(t *testing.T) {
	rri := &RouteRuleImplBase{
		prefixRewrite: "/abc/",
	}
	type args struct {
		headers     types.HeaderMap
		matchedPath string
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/"},
				matchedPath: "/",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/abc/", protocol.MosnOriginalHeaderPathKey: "/"},
		},
		{
			name: "case2",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/index/page/"},
				matchedPath: "/index/",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/abc/page/", protocol.MosnOriginalHeaderPathKey: "/index/page/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rri.finalizePathHeader(tt.args.headers, tt.args.matchedPath)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(rri *RouteRuleImplBase) finalizePathHeader(headers map[string]string, matchedPath string) = %v, want %v", tt.args.headers, tt.want)
			}
		})

	}
}

func Test_RouteRuleImplBase_FinalizeRequestHeaders(t *testing.T) {

	type args struct {
		rri         *RouteRuleImplBase
		headers     types.HeaderMap
		requestInfo types.RequestInfo
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				rri: &RouteRuleImplBase{
					hostRewrite: "www.xxx.com",
					requestHeadersParser: &headerParser{
						headersToAdd: []*headerPair{
							{
								headerName: &lowerCaseString{"level"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "1",
								},
							},
							{
								headerName: &lowerCaseString{"route"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "true",
								},
							},
						},
					},
					vHost: &VirtualHostImpl{
						requestHeadersParser: &headerParser{
							headersToAdd: []*headerPair{
								{
									headerName: &lowerCaseString{"level"},
									headerFormatter: &plainHeaderFormatter{
										isAppend:    true,
										staticValue: "2",
									},
								},
								{
									headerName: &lowerCaseString{"vhost"},
									headerFormatter: &plainHeaderFormatter{
										isAppend:    true,
										staticValue: "true",
									},
								},
							},
						},
						globalRouteConfig: &configImpl{
							requestHeadersParser: &headerParser{
								headersToAdd: []*headerPair{
									{
										headerName: &lowerCaseString{"level"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "3",
										},
									},
									{
										headerName: &lowerCaseString{"global"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "true",
										},
									},
								},
							},
						},
					},
				},
				headers:     protocol.CommonHeader{"host": "xxx.default.svc.cluster.local"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"host": "xxx.default.svc.cluster.local", "authority": "www.xxx.com", "level": "1,2,3", "route": "true", "vhost": "true", "global": "true"},
		},

		{
			name: "case2",
			args: args{
				rri: &RouteRuleImplBase{
					requestHeadersParser: &headerParser{
						headersToAdd: []*headerPair{
							{
								headerName: &lowerCaseString{"level"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "1",
								},
							},
							{
								headerName: &lowerCaseString{"route"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "true",
								},
							},
						},
					},
					vHost: &VirtualHostImpl{
						globalRouteConfig: &configImpl{
							requestHeadersParser: &headerParser{
								headersToAdd: []*headerPair{
									{
										headerName: &lowerCaseString{"level"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "3",
										},
									},
									{
										headerName: &lowerCaseString{"global"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "true",
										},
									},
								},
							},
						},
					},
				},
				headers:     protocol.CommonHeader{"host": "xxx.default.svc.cluster.local"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"host": "xxx.default.svc.cluster.local", "level": "1,3", "route": "true", "global": "true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.rri.FinalizeRequestHeaders(tt.args.headers, tt.args.requestInfo)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(rri *RouteRuleImplBase) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) = %v, want %v", tt.args.headers, tt.want)
			}
		})
	}
}

func Test_RouteRuleImplBase_FinalizeResponseHeaders(t *testing.T) {

	type args struct {
		rri         *RouteRuleImplBase
		headers     types.HeaderMap
		requestInfo types.RequestInfo
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				rri: &RouteRuleImplBase{
					responseHeadersParser: &headerParser{
						headersToAdd: []*headerPair{
							{
								headerName: &lowerCaseString{"level"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "1",
								},
							},
							{
								headerName: &lowerCaseString{"route"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "true",
								},
							},
						},
						headersToRemove: []*lowerCaseString{{"status"}, {"username"}},
					},
					vHost: &VirtualHostImpl{
						responseHeadersParser: &headerParser{
							headersToAdd: []*headerPair{
								{
									headerName: &lowerCaseString{"level"},
									headerFormatter: &plainHeaderFormatter{
										isAppend:    true,
										staticValue: "2",
									},
								},
								{
									headerName: &lowerCaseString{"vhost"},
									headerFormatter: &plainHeaderFormatter{
										isAppend:    true,
										staticValue: "true",
									},
								},
							},
							headersToRemove: []*lowerCaseString{{"ver"}},
						},
						globalRouteConfig: &configImpl{
							responseHeadersParser: &headerParser{
								headersToAdd: []*headerPair{
									{
										headerName: &lowerCaseString{"level"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "3",
										},
									},
									{
										headerName: &lowerCaseString{"global"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "true",
										},
									},
								},
								headersToRemove: []*lowerCaseString{{"x-mosn"}},
							},
						},
					},
				},
				headers:     protocol.CommonHeader{"status": "ready", "username": "xx", "ver": "0.1", "x-mosn": "100"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"level": "1,2,3", "route": "true", "vhost": "true", "global": "true"},
		},
		{
			name: "case2",
			args: args{
				rri: &RouteRuleImplBase{
					responseHeadersParser: &headerParser{
						headersToAdd: []*headerPair{
							{
								headerName: &lowerCaseString{"level"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "1",
								},
							},
							{
								headerName: &lowerCaseString{"route"},
								headerFormatter: &plainHeaderFormatter{
									isAppend:    true,
									staticValue: "true",
								},
							},
						},
						headersToRemove: []*lowerCaseString{{"status"}, {"username"}},
					},
					vHost: &VirtualHostImpl{
						globalRouteConfig: &configImpl{
							responseHeadersParser: &headerParser{
								headersToAdd: []*headerPair{
									{
										headerName: &lowerCaseString{"level"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "3",
										},
									},
									{
										headerName: &lowerCaseString{"global"},
										headerFormatter: &plainHeaderFormatter{
											isAppend:    true,
											staticValue: "true",
										},
									},
								},
								headersToRemove: []*lowerCaseString{{"x-mosn"}},
							},
						},
					},
				},
				headers:     protocol.CommonHeader{"status": "ready", "username": "xx", "ver": "0.1", "x-mosn": "100"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"ver": "0.1", "level": "1,3", "route": "true", "global": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.rri.FinalizeResponseHeaders(tt.args.headers, tt.args.requestInfo)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(rri *RouteRuleImplBase) FinalizeResponseHeaders(headers map[string]string, requestInfo types.RequestInfo) = %v, want %v", tt.args.headers, tt.want)
			}
		})
	}
}
