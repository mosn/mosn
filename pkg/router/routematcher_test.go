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
	"strings"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

func newTestSimpleRouter(name string) v2.Router {
	r := v2.Router{}
	r.Match = v2.RouterMatch{
		Headers: []v2.HeaderMatcher{
			v2.HeaderMatcher{Name: "service", Value: ".*"},
		},
	}
	r.Route = v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName: name,
		},
	}
	return r
}

var testVirutalHostConfigs = map[string]*v2.VirtualHost{
	"all":             &v2.VirtualHost{Name: "all", Domains: []string{"*"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
	"wildcard-domain": &v2.VirtualHost{Name: "wildcard-domain", Domains: []string{"*.sofa-mosn.test"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
	"domain":          &v2.VirtualHost{Name: "domain", Domains: []string{"www.sofa-mosn.test"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
}

func TestNewRouteMatcherSingle(t *testing.T) {
	// Single VirtualHost
	// Add VirtualHost type verify
	testCases := []struct {
		virtualHost *v2.VirtualHost
		Expected    func(*routeMatcher) bool
	}{
		{
			virtualHost: testVirutalHostConfigs["all"],
			Expected: func(rm *routeMatcher) bool {
				return rm.defaultVirtualHost != nil
			},
		},
		{
			virtualHost: testVirutalHostConfigs["wildcard-domain"],
			Expected: func(rm *routeMatcher) bool {
				return len(rm.wildcardVirtualHostSuffixes) == 1
			},
		},
		{
			virtualHost: testVirutalHostConfigs["domain"],
			Expected: func(rm *routeMatcher) bool {
				return len(rm.virtualHosts) == 1
			},
		},
	}
	for _, tc := range testCases {

		cfg := &v2.RouterConfiguration{
			VirtualHosts: []*v2.VirtualHost{tc.virtualHost},
		}

		routers, err := NewRouteMatcher(cfg)
		if err != nil {
			t.Errorf("#%s : %v\n", tc.virtualHost.Name, err)
			continue
		}
		rm := routers.(*routeMatcher)
		if !tc.Expected(rm) {
			t.Errorf("#%s : expected one virtualhost\n", tc.virtualHost.Name)
		}
	}
}

func TestNewRouteMatcherGroup(t *testing.T) {
	//A group of VirtualHost
	var virtualhosts []*v2.VirtualHost
	for _, vhConfig := range testVirutalHostConfigs {
		virtualhosts = append(virtualhosts, vhConfig)
	}
	cfg := &v2.RouterConfiguration{
		VirtualHosts: virtualhosts,
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	rm := routers.(*routeMatcher)
	expected := rm.defaultVirtualHost != nil && len(rm.virtualHosts) == 1 && len(rm.wildcardVirtualHostSuffixes) == 1
	if !expected {
		t.Error("create routematcher not match")
	}
}

func TestNewRouteMatcherDuplicate(t *testing.T) {
	// two virtualhosts, both domain is "*", expected failed
	if _, err := NewRouteMatcher(&v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["all"], testVirutalHostConfigs["all"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	//two virtualhosts, both domain is "www.sofa-mosn.test", expected failed
	if _, err := NewRouteMatcher(&v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["domain"], testVirutalHostConfigs["domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with same suffix, expected failed
	if _, err := NewRouteMatcher(&v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["wildcard-domain"], testVirutalHostConfigs["wildcard-domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with different suffix:
	// *.test.com, *.test.net, *.test.com.cn
	// expected OK
	if _, err := NewRouteMatcher(&v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Domains: []string{"*.test.com"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
			&v2.VirtualHost{Domains: []string{"*.test.net"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
			&v2.VirtualHost{Domains: []string{"*.test.com.cn"}, Routers: []v2.Router{newTestSimpleRouter("test")}},
		},
	}); err != nil {
		t.Error("NewRouteMatcher with different wildcard domain failed")
	}

}

// match all
func TestDefaultMatch(t *testing.T) {
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			testVirutalHostConfigs["all"],
		},
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("create router matcher failed %v\n", err)
		return
	}
	testCases := []string{
		"*.test.com",
		"*",
		"foo.com",
		"12345678",
	}
	for i, tc := range testCases {
		headers := protocol.CommonHeader(map[string]string{
			strings.ToLower(protocol.MosnHeaderHostKey): tc,
			"service": "test",
		})
		if routers.Route(headers, 1) == nil {
			t.Errorf("#%d not matched\n", i)
		}
		if routers.GetAllRoutes(headers, 1) == nil {
			t.Errorf("#%d not matched\n", i)
		}
	}
}
func TestDomainMatch(t *testing.T) {
	cfg := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			testVirutalHostConfigs["domain"],
		},
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("create router matcher failed %v\n", err)
		return
	}
	headers := protocol.CommonHeader(map[string]string{
		strings.ToLower(protocol.MosnHeaderHostKey): "www.sofa-mosn.test",
		"service": "test",
	})
	if routers.Route(headers, 1) == nil {
		t.Error("domain match failed")
	}
	if routers.GetAllRoutes(headers, 1) == nil {
		t.Error("domain match failed")
	}
	//not matched
	notMatched := []string{
		"*",
		"*www.sofa-mosn.test",
		"sofa-mosn.test",
		"www.sofa-mosn",
		"www.sofa-mosn.test1",
		"*.sofa-mosn.test",
	}
	for i, tc := range notMatched {
		headers := protocol.CommonHeader(map[string]string{
			strings.ToLower(protocol.MosnHeaderHostKey): tc,
			"service": "test",
		})
		if routers.Route(headers, 1) != nil {
			t.Errorf("#%d expected not matched, but match a router", i)
		}
		if routers.GetAllRoutes(headers, 1) != nil {
			t.Errorf("#%d expected not matched, but match a router", i)
		}
	}
}
func TestWildcardMatch(t *testing.T) {
	testCases := []struct {
		wildcardDomain  string
		matchedDomain   []string
		unmatchedDomain []string
	}{
		{
			wildcardDomain:  "*.test.com",
			matchedDomain:   []string{"a.test.com", "a.test.test.com", "abc.test.com", "abc-def.test.com", "a.b.test.com", "*.test.com"},
			unmatchedDomain: []string{"a.test.net", "a-test.com", ".test.com", "*"},
		},
		{
			wildcardDomain:  "*.test.net",
			matchedDomain:   []string{"a.test.net"},
			unmatchedDomain: []string{"a.test.com"},
		},
		{
			wildcardDomain:  "*.test.com.cn",
			matchedDomain:   []string{"a.test.com.cn"},
			unmatchedDomain: []string{"a.test.com", "a.test.cn", "a.aaaa.com.cn"},
		},
		{
			wildcardDomain:  "*-bar.foo.com",
			matchedDomain:   []string{"a-bar.foo.com", "a.b-bar.foo.com", "a.-bar.foo.com"},
			unmatchedDomain: []string{"a.foo.com", "*.foo.com", "bar.foo.com", "-bar.test.com"},
		},
	}
	simpleRouter := newTestSimpleRouter("testRouter")
	for i, tc := range testCases {
		vh := &v2.VirtualHost{
			Domains: []string{tc.wildcardDomain},
			Routers: []v2.Router{simpleRouter},
		}
		cfg := &v2.RouterConfiguration{
			VirtualHosts: []*v2.VirtualHost{vh},
		}
		routers, err := NewRouteMatcher(cfg)
		if err != nil {
			t.Errorf("#%d create routers failed: %v\n", i, err)
			continue
		}
		for _, match := range tc.matchedDomain {
			headers := protocol.CommonHeader(map[string]string{
				strings.ToLower(protocol.MosnHeaderHostKey): match,
				"service": "test",
			})
			if routers.Route(headers, 1) == nil {
				t.Errorf("%s expected matched: #%d, but return nil\n", match, i)
			}
			if routers.GetAllRoutes(headers, 1) == nil {
				t.Errorf("%s expected matched: #%d, but return nil\n", match, i)
			}
		}
		for _, unmatch := range tc.unmatchedDomain {
			headers := protocol.CommonHeader(map[string]string{
				strings.ToLower(protocol.MosnHeaderHostKey): unmatch,
				"service": "test",
			})
			if routers.Route(headers, 1) != nil {
				t.Errorf("%s expected unmatched: #%d, but matched\n", unmatch, i)
			}
			if routers.GetAllRoutes(headers, 1) != nil {
				t.Errorf("%s expected unmatched: #%d, but matched\n", unmatch, i)
			}
		}
	}

}

func TestWildcardLongestSuffixMatch(t *testing.T) {
	virtualHosts := []*v2.VirtualHost{
		&v2.VirtualHost{Domains: []string{"f-bar.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("domain")}},
		&v2.VirtualHost{Domains: []string{"*.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("short")}},
		&v2.VirtualHost{Domains: []string{"*-bar.baz.com"}, Routers: []v2.Router{newTestSimpleRouter("long")}},
		&v2.VirtualHost{Domains: []string{"*.foo.com"}, Routers: []v2.Router{newTestSimpleRouter("foo")}},
	}
	cfg := &v2.RouterConfiguration{
		VirtualHosts: virtualHosts,
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	testCases := []struct {
		Domain        string
		ExpectedRoute string
	}{
		{Domain: "f-bar.baz.com", ExpectedRoute: "domain"},
		{Domain: "foo-bar.baz.com", ExpectedRoute: "long"},
		{Domain: "foo.baz.com", ExpectedRoute: "short"},
		{Domain: "foo.foo.com", ExpectedRoute: "foo"},
	}
	for _, tc := range testCases {
		route := routers.Route(protocol.CommonHeader(map[string]string{
			strings.ToLower(protocol.MosnHeaderHostKey): tc.Domain,
			"service": "test",
		}), 1)
		if route == nil {
			t.Errorf("%s match failed\n", tc.Domain)
			continue
		}
		if route.RouteRule().ClusterName() != tc.ExpectedRoute {
			t.Errorf("%s expected match %s, but got %s\n", tc.Domain, tc.ExpectedRoute, route.RouteRule().ClusterName())
		}
	}
}
func TestInvalidConfig(t *testing.T) {
	var testCases []*v2.RouterConfiguration
	//1. invalid config object
	testCases = append(testCases, nil)
	//2. config without router
	case2 := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Domains: []string{"*"}},
		},
	}
	//3. config without matcher
	case3 := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Domains: []string{"*"}, Routers: []v2.Router{
				v2.Router{
					RouterConfig: v2.RouterConfig{
						Route: v2.RouteAction{
							RouterActionConfig: v2.RouterActionConfig{
								ClusterName: "www",
							},
						},
					},
				},
			}},
		},
	}
	//4. an invalid regexp matcher
	case4 := &v2.RouterConfiguration{
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Domains: []string{"*"}, Routers: []v2.Router{
				v2.Router{
					RouterConfig: v2.RouterConfig{
						Match: v2.RouterMatch{Regex: "/f["},
						Route: v2.RouteAction{
							RouterActionConfig: v2.RouterActionConfig{
								ClusterName: "www",
							},
						},
					},
				}},
			}},
	}
	testCases = append(testCases, case2, case3, case4)
	for i, tc := range testCases {
		if _, err := NewRouteMatcher(tc); err == nil {
			t.Errorf("#%d expected error, but not", i)
		}
	}
}

func TestMain(m *testing.M) {
	log.InitDefaultLogger("", log.DEBUG)
	m.Run()
}
