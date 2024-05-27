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
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/pkg/variable"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	mhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func TestHTTPRuleMatchMethod(t *testing.T) {
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

	httpRule := NewBaseHTTPRouteRule(routeRuleBase, route.Match.Headers)

	headers := mhttp.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	ctx := variable.NewVariableContext(context.Background())
	variable.SetString(ctx, types.VarMethod, "POST")
	match := httpRule.matchRoute(ctx, headers)
	if !assert.Truef(t, match, "match http method failed, result should be true, get %+v", match) {
		t.FailNow()
	}

	http2Request := &http.Request{
		Method: "POST",
		Header: http.Header{},
	}
	headerHttp2 := http2.NewReqHeader(http2Request)
	match = httpRule.matchRoute(ctx, headerHttp2)
	if !assert.Truef(t, match, "match http2 method failed, result should be true, get %+v", match) {
		t.FailNow()
	}

}

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
	ctx := variable.NewVariableContext(context.Background())
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
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		rr := &PrefixRouteRuleImpl{
			NewBaseHTTPRouteRule(base, nil),
			route.Match.Prefix,
		}
		headers := protocol.CommonHeader(map[string]string{})
		variable.SetString(ctx, types.VarPath, tc.headerpath)
		result := rr.Match(ctx, headers)
		if (result != nil) != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
		if result != nil {
			if result.RouteRule().PathMatchCriterion().MatchType() != api.Prefix {
				t.Errorf("#%d match type is not expected", i)
			}
		}
	}
}

func TestPathRouteRuleImpl(t *testing.T) {
	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		path       string
		headerpath string
		expected   bool
	}{
		{"/test", "/test", true},
		{"/test", "/Test", true},
		{"/test", "/test/test", false},
	}
	ctx := variable.NewVariableContext(context.Background())
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
		rr := &PathRouteRuleImpl{NewBaseHTTPRouteRule(base, nil), route.Match.Path}
		headers := protocol.CommonHeader(map[string]string{})
		variable.SetString(ctx, types.VarPath, tc.headerpath)
		result := rr.Match(ctx, headers)
		if (result != nil) != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
		if result != nil {
			if result.RouteRule().PathMatchCriterion().MatchType() != api.Exact {
				t.Errorf("#%d match type is not expected", i)
			}
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
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)

		rr := &RegexRouteRuleImpl{
			NewBaseHTTPRouteRule(base, nil),
			route.Match.Regex,
			re,
		}
		ctx := variable.NewVariableContext(context.Background())
		headers := protocol.CommonHeader(map[string]string{})
		variable.SetString(ctx, types.VarPath, tc.headerpath)
		result := rr.Match(ctx, headers)
		if (result != nil) != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
		if result != nil {
			if result.RouteRule().PathMatchCriterion().MatchType() != api.Regex {
				t.Errorf("#%d match type is not expected", i)
			}
		}
	}
}
