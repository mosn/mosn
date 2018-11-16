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
	"reflect"
	"regexp"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestPrefixRouteRuleImpl(t *testing.T) {
	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		prefix     string
		headerpath string
		expected   bool
	}{
		{"/", "/", true},
		{"/", "/test", true},
		{"/", "/test/foo", true},
		{"/", "/foo?key=value", true},
		{"/foo", "/foo", true},
		{"/foo", "/footest", true},
		{"/foo", "/foo/test", true},
		{"/foo", "/foo?key=value", true},
		{"/foo", "/", false},
		{"/foo", "/test", false},
	}
	for i, tc := range testCases {
		route := &v2.Router{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{Prefix: tc.prefix},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: "test",
					},
				},
			},
		}
		routuRule, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		rr := &PrefixRouteRuleImpl{
			routuRule,
			route.Match.Prefix,
		}
		headers := protocol.CommonHeader(map[string]string{protocol.MosnHeaderPathKey: tc.headerpath})
		result := rr.Match(headers, 1) != nil
		if result != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
	}
}

func TestPathRouteRuleImpl(t *testing.T) {
	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		path          string
		headerpath    string
		caseSensitive bool //no interface to set caseSensitive, need hack
		expected      bool
	}{
		{"/test", "/test", false, true},
		{"/test", "/Test", false, true},
		{"/test", "/Test", true, false},
		{"/test", "/test/test", false, false},
	}
	for i, tc := range testCases {
		route := &v2.Router{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{Path: tc.path},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: "test",
					},
				},
			},
		}
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		base.caseSensitive = tc.caseSensitive //hack case sensitive
		rr := &PathRouteRuleImpl{base, route.Match.Path}
		headers := protocol.CommonHeader(map[string]string{protocol.MosnHeaderPathKey: tc.headerpath})
		result := rr.Match(headers, 1) != nil
		if result != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}

	}
}

func TestRegexRouteRuleImpl(t *testing.T) {
	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		regexp     string
		headerpath string
		expected   bool
	}{
		{".*", "/", true},
		{".*", "/path", true},
		{"/[0-9]+", "/12345", true},
		{"/[0-9]+", "/test", false},
	}
	for i, tc := range testCases {
		route := &v2.Router{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{Regex: tc.regexp},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: "test",
					},
				},
			},
		}
		re := regexp.MustCompile(tc.regexp)
		routuRule, _ := NewRouteRuleImplBase(virtualHostImpl, route)

		rr := &RegexRouteRuleImpl{
			routuRule,
			route.Match.Regex,
			re,
		}
		headers := protocol.CommonHeader(map[string]string{protocol.MosnHeaderPathKey: tc.headerpath})
		result := rr.Match(headers, 1) != nil
		if result != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
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
						MetadataMatch: v2.Metadata{
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
						MetadataMatch: v2.Metadata{
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
						MetadataMatch: v2.Metadata{
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
						MetadataMatch: v2.Metadata{
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

func TestRouteRuleImplBase_UpdateMetaDataMatchCriteria(t *testing.T) {
	originMetadatas := []struct {
		originMetadata map[string]string
	}{
		{
			originMetadata: map[string]string{"origin": "test"},
		},
		{
			originMetadata: map[string]string{},
		},
	}

	updatedMetadatas := []struct {
		updatedMetadata map[string]string
	}{
		{
			updatedMetadata: map[string]string{
				"label": "green", "version": "v1", "appInfo": "test",
			},
		},
		{
			updatedMetadata: map[string]string{},
		},
	}

	tests := []struct {
		name            string
		updatedMetadata map[string]string
		origin          *RouteRuleImplBase
		want            *RouteRuleImplBase
	}{
		{
			name: "common case",
			origin: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(originMetadatas[0].originMetadata),
				metaData:              getClusterMosnLBMetaDataMap(originMetadatas[0].originMetadata),
			},
			updatedMetadata: updatedMetadatas[0].updatedMetadata,
			want: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(updatedMetadatas[0].updatedMetadata),
				metaData:              getClusterMosnLBMetaDataMap(updatedMetadatas[0].updatedMetadata),
			},
		},
		{
			name: "corner case1",
			origin: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(originMetadatas[0].originMetadata),
				metaData:              getClusterMosnLBMetaDataMap(originMetadatas[0].originMetadata),
			},
			updatedMetadata: updatedMetadatas[1].updatedMetadata,
			want: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(updatedMetadatas[1].updatedMetadata),
				metaData:              getClusterMosnLBMetaDataMap(updatedMetadatas[1].updatedMetadata),
			},
		},
		{
			name: "corner case2",
			origin: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(originMetadatas[1].originMetadata),
				metaData:              getClusterMosnLBMetaDataMap(originMetadatas[1].originMetadata),
			},
			updatedMetadata: updatedMetadatas[1].updatedMetadata,
			want: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(updatedMetadatas[0].updatedMetadata),
				metaData:              getClusterMosnLBMetaDataMap(updatedMetadatas[0].updatedMetadata),
			},
		},
		{
			name: "corner case3",
			origin: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(originMetadatas[1].originMetadata),
				metaData:              getClusterMosnLBMetaDataMap(originMetadatas[1].originMetadata),
			},
			updatedMetadata: updatedMetadatas[1].updatedMetadata,
			want: &RouteRuleImplBase{
				metadataMatchCriteria: NewMetadataMatchCriteriaImpl(updatedMetadatas[1].updatedMetadata),
				metaData:              getClusterMosnLBMetaDataMap(updatedMetadatas[1].updatedMetadata),
			},
		},
	}

	for _, tt := range tests {
		if reflect.DeepEqual(tt.origin.UpdateMetaDataMatchCriteria(tt.updatedMetadata), tt.want) {
			t.Errorf("TestRouteRuleImplBase_UpdateMetaDataMatchCriteria,error!")
		}

	}
}
