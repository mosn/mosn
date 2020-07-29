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
	"context"
	"fmt"
	"math/rand"
	"net"
	goHttp "net/http"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
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

	//both prefix_rewrite and regex_rewrite are configured, prefix rewrite by default
	rri := &RouteRuleImplBase{
		prefixRewrite: "/abc/",
		regexRewrite: v2.RegexRewrite{
			Pattern: v2.PatternConfig{
				Regex: "^/service/([^/]+)(/.*)$",
			},
			Substitution: "${2}/instance/${1}",
		},
	}

	regexPattern, err := regexp.Compile(rri.regexRewrite.Pattern.Regex)
	assert.NoErrorf(t, err, "check regrexp pattern failed %+v", err)
	rri.regexPattern = regexPattern

	type args struct {
		headers     types.HeaderMap
		matchedPath string
	}

	type testCase struct {
		name string
		args args
		want types.HeaderMap
	}

	tests := []testCase{
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
			rri.FinalizePathHeader(tt.args.headers, tt.args.matchedPath)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(rri *RouteRuleImplBase) finalizePathHeader(headers map[string]string, matchedPath string) = %v, want %v", tt.args.headers, tt.want)
			}
		})

	}

	//regex rewrite test
	rris := []*RouteRuleImplBase{
		{
			regexRewrite: v2.RegexRewrite{
				Pattern: v2.PatternConfig{
					Regex: "^/service/([^/]+)(/.*)$",
				},
				Substitution: "${2}/instance/${1}",
			},
		},
		{
			regexRewrite: v2.RegexRewrite{
				Pattern: v2.PatternConfig{
					Regex: "one",
				},
				Substitution: "two",
			},
		},
		{
			regexRewrite: v2.RegexRewrite{
				Pattern: v2.PatternConfig{
					Regex: "^(.*?)one(.*)$",
				},
				Substitution: "${1}two${2}",
			},
		},
		{
			regexRewrite: v2.RegexRewrite{
				Pattern: v2.PatternConfig{
					Regex: "(?i)/xxx/",
				},
				Substitution: "/yyy/",
			},
		},
	}

	tests = []testCase{
		{
			name: "case1",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/service/foo/v1/api"},
				matchedPath: "/service/foo/v1/api",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/v1/api/instance/foo", protocol.MosnOriginalHeaderPathKey: "/service/foo/v1/api"},
		},
		{
			name: "case2",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/xxx/one/yyy/one/zzz"},
				matchedPath: "/xxx/one/yyy/one/zzz",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/xxx/two/yyy/two/zzz", protocol.MosnOriginalHeaderPathKey: "/xxx/one/yyy/one/zzz"},
		},
		{
			name: "case3",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/xxx/one/yyy/one/zzz"},
				matchedPath: "/xxx/one/yyy/one/zzz",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/xxx/two/yyy/one/zzz", protocol.MosnOriginalHeaderPathKey: "/xxx/one/yyy/one/zzz"},
		},
		{
			name: "case4",
			args: args{
				headers:     protocol.CommonHeader{protocol.MosnHeaderPathKey: "/aaa/XxX/bbb"},
				matchedPath: "/aaa/XxX/bbb",
			},
			want: protocol.CommonHeader{protocol.MosnHeaderPathKey: "/aaa/yyy/bbb", protocol.MosnOriginalHeaderPathKey: "/aaa/XxX/bbb"},
		},
	}

	testMap := make(map[string]testCase)
	for _, tt := range tests {
		testMap[tt.name] = tt
	}

	for k, rri := range rris {

		regexPattern, err := regexp.Compile(rri.regexRewrite.Pattern.Regex)
		assert.NoErrorf(t, err, "check regrexp pattern failed %+v", err)
		rri.regexPattern = regexPattern

		ops := k
		tt, ok := testMap["case"+strconv.Itoa(k+1)]
		if ok {
			t.Run(tt.name, func(t *testing.T) {
				rris[ops].FinalizePathHeader(tt.args.headers, tt.args.matchedPath)
				if !reflect.DeepEqual(tt.args.headers, tt.want) {
					t.Errorf("(rri *RouteRuleImplBase) finalizePathHeader(headers map[string]string, matchedPath string) = %v, want %v", tt.args.headers, tt.want)
				}
			})
		}
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

		{
			name: "case3",
			args: args{
				rri: &RouteRuleImplBase{
					// defaultCluster: &weightedClusterEntry{
					// 	clusterName: "case3Cluster",
					// },
					autoHostRewriteHeader: "realyHost",
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
				headers:     protocol.CommonHeader{"realyHost": "mosn.io.rewrited.host"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"realyHost": "mosn.io.rewrited.host", "authority": "mosn.io.rewrited.host", "level": "1,3", "route": "true", "global": "true"},
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

func TestParseHashPolicy(t *testing.T) {
	routerMock1 := &v2.Router{}
	routerMock1.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: "defaultCluster",
			HashPolicy: []v2.HashPolicy{
				{
					Header: &v2.HeaderHashPolicy{Key: "header_key"},
				},
				// test parse first hash policy
				{
					SourceIP: &v2.SourceIPHashPolicy{},
				},
			},
		},
	}

	rb, err := NewRouteRuleImplBase(nil, routerMock1)
	assert.NoErrorf(t, err, "new routerule impl failed %+v", err)
	headerHp, ok := rb.policy.hashPolicy.(*headerHashPolicyImpl)
	assert.Truef(t, ok, "hash policy should be headerHashPolicyImpl type")
	if ok {
		assert.Equalf(t, "header_key", headerHp.key,
			"headerHashPolicyImpl key should be 'header_key'")
	}

	// test parse each type of hash policy
	// sourceIP
	routerMock1.Route.HashPolicy = []v2.HashPolicy{
		{
			SourceIP: &v2.SourceIPHashPolicy{},
		},
	}
	rb, err = NewRouteRuleImplBase(nil, routerMock1)
	assert.NoErrorf(t, err, "new routerule impl failed %+v", err)
	_, ok = rb.policy.hashPolicy.(*sourceIPHashPolicyImpl)
	assert.Truef(t, ok, "hash policy should be sourceIPHashPolicyImpl type")

	// httpCookie
	routerMock1.Route.HashPolicy = []v2.HashPolicy{
		{
			Cookie: &v2.CookieHashPolicy{
				Name: "cookie_name",
				Path: "cookie_path",
				TTL: api.DurationConfig{
					5 * time.Second,
				},
			},
		},
	}
	rb, err = NewRouteRuleImplBase(nil, routerMock1)
	assert.NoErrorf(t, err, "new routerule impl failed %+v", err)
	cookieHp, ok := rb.policy.hashPolicy.(*cookieHashPolicyImpl)
	assert.Truef(t, ok, "hash policy should be httpCookieHashPolicyImpl type")
	if ok {
		assert.Equalf(t, "cookie_name", cookieHp.name,
			"httpCookieHashPolicyImpl key should be 'cookie_name'")
		assert.Equalf(t, "cookie_path", cookieHp.path,
			"httpCookieHashPolicyImpl key should be 'cookie_name'")
		assert.Equalf(t, 5*time.Second, cookieHp.ttl.Duration,
			"httpCookieHashPolicyImpl key should be '5s'")
	}

}

func TestHashPolicy(t *testing.T) {
	testProtocol := types.ProtocolName("SomeProtocol")
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamProtocol, testProtocol)

	// test header
	headerGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_header_value", nil
	}
	headerValue := variable.NewBasicVariable("SomeProtocol_request_header_", nil, headerGetter, nil, 0)
	variable.RegisterPrefixVariable(headerValue.Name(), headerValue)
	variable.RegisterProtocolResource(testProtocol, api.HEADER, types.VarProtocolRequestHeader)
	headerHp := headerHashPolicyImpl{
		key: "header_key",
	}
	hash := headerHp.GenerateHash(ctx)
	assert.Equalf(t, uint64(3684553712465070601), hash, "header value hash not match")

	// test cookie
	cookieGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_cookie_value", nil
	}
	cookieValue := variable.NewBasicVariable("SomeProtocol_cookie_", nil, cookieGetter, nil, 0)
	variable.RegisterPrefixVariable(cookieValue.Name(), cookieValue)
	variable.RegisterProtocolResource(testProtocol, api.COOKIE, types.VarProtocolCookie)
	cookieHp := cookieHashPolicyImpl{
		name: "cookie_name",
		path: "cookie_path",
		ttl:  api.DurationConfig{5 * time.Second},
	}
	hash = cookieHp.GenerateHash(ctx)
	assert.Equalf(t, uint64(14068947270705736519), hash, "cookie value hash not match")

	// test source IP
	ctx = mosnctx.WithValue(ctx, types.ContextOriRemoteAddr, &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 80,
	})
	sourceIPHp := sourceIPHashPolicyImpl{}
	hash = sourceIPHp.GenerateHash(ctx)
	assert.Equalf(t, uint64(2130706433), hash, "source ip hash not match")
}

// TestDefaultHashPolicy tests use sourceIPHashPolicy as default hash policy
func TestDefaultHashPolicy(t *testing.T) {
	routerMock1 := &v2.Router{}
	routerMock1.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: "defaultCluster",
			// nil HashPolicy field
		},
	}

	rb, err := NewRouteRuleImplBase(nil, routerMock1)
	assert.NoErrorf(t, err, "err should be nil, but get %+v", err)
	assert.IsTypef(t, rb.policy.HashPolicy(), &sourceIPHashPolicyImpl{}, "")
}

func TestRedirectRule(t *testing.T) {
	testCases := []struct {
		name           string
		redirectAction *v2.RedirectAction
		expected       *redirectImpl
		expectErr      error
	}{
		{
			name: "invalid response code",
			redirectAction: &v2.RedirectAction{
				ResponseCode: 400,
				PathRedirect: "/foo",
			},
			expectErr: fmt.Errorf("redirect code not supported yet: 400"),
		},
		{
			name: "invalid scheme",
			redirectAction: &v2.RedirectAction{
				SchemeRedirect: "1http",
			},
			expectErr: fmt.Errorf("invalid scheme: 1http"),
		},
		{
			name: "path redirect",
			redirectAction: &v2.RedirectAction{
				PathRedirect: "/foo",
			},
			expected: &redirectImpl{
				path: "/foo",
				code: goHttp.StatusMovedPermanently,
			},
		},
		{
			name: "host redirect",
			redirectAction: &v2.RedirectAction{
				HostRedirect: "foo.com",
				ResponseCode: goHttp.StatusTemporaryRedirect,
			},
			expected: &redirectImpl{
				host: "foo.com",
				code: goHttp.StatusTemporaryRedirect,
			},
		},
		{
			name: "scheme redirect",
			redirectAction: &v2.RedirectAction{
				SchemeRedirect: "https",
			},
			expected: &redirectImpl{
				scheme: "https",
				code:   goHttp.StatusMovedPermanently,
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			rb, err := NewRouteRuleImplBase(nil, &v2.Router{
				RouterConfig: v2.RouterConfig{
					Redirect: tc.redirectAction,
				},
			})
			if tc.expectErr != nil {
				if err == nil {
					t.Errorf("Unexpected success")
					return
				}
				if err.Error() != tc.expectErr.Error() {
					t.Errorf("Expect error: %s\nGot: %s\n", tc.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
				return
			}
			if !reflect.DeepEqual(rb.redirectRule, tc.expected) {
				t.Errorf("Unexpected redirect rule\nExpected: %#v\nGot: %#v\n", *tc.expected, *rb.redirectRule)
			}
		})
	}
}
