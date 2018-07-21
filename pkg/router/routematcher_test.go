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

	"github.com/alipay/sofa-mosn/internal/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

var testVirutalHostConfigs = map[string]*v2.VirtualHost{
	"all":             &v2.VirtualHost{Name: "all", Domains: []string{"*"}},
	"wildcard-domain": &v2.VirtualHost{Name: "wildcard-domain", Domains: []string{"*.sofa-mosn.test"}},
	"domain":          &v2.VirtualHost{Name: "domain", Domains: []string{"www.sofa-mosn.test"}},
}

var testInvalidVirutalHostConfigs = map[string]*v2.VirtualHost{
	"inall":             &v2.VirtualHost{Name: "all", Domains: []string{"t*t"}},
	"inwildcard-domain": &v2.VirtualHost{Name: "wildcard-domain", Domains: []string{".*.sofa-mosn.test"}},
	"indomain":          &v2.VirtualHost{Name: "domain", Domains: []string{"*www.sofa-mosn.test"}},
}

func TestNewRouteMatcher_default(t *testing.T) {
	
	virtualhost := testVirutalHostConfigs["all"]
	
	cfg := &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm := routers.(*routeMatcher)
	expected := (rm.defaultVirtualHost != nil)
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
	
	virtualhost = testInvalidVirutalHostConfigs["inall"]
	
	cfg = &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err = NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm = routers.(*routeMatcher)
	expected = (rm.defaultVirtualHost == nil)
	
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
}

func TestNewRouteMatcher_wildcard(t *testing.T) {
	
	virtualhost := testVirutalHostConfigs["wildcard-domain"]
	
	cfg := &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm := routers.(*routeMatcher)
	expected := (len(rm.wildcardVirtualHostSuffixes) == 1)
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
	
	virtualhost = testInvalidVirutalHostConfigs["inwildcard-domain"]
	
	cfg = &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err = NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm = routers.(*routeMatcher)
	expected = (len(rm.wildcardVirtualHostSuffixes) == 0)
	
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
}

func TestNewRouteMatcher_domain(t *testing.T) {
	
	virtualhost := testVirutalHostConfigs["domain"]
	
	cfg := &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm := routers.(*routeMatcher)
	expected := (len(rm.virtualHosts) == 1)
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
	
	virtualhost = testInvalidVirutalHostConfigs["indomain"]
	
	cfg = &v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{virtualhost},
	}
	routers, err = NewRouteMatcher(cfg)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	
	rm = routers.(*routeMatcher)
	expected = (len(rm.virtualHosts) == 0)
	
	if !expected {
		t.Errorf("#%s : expected one virtualhost\n")
	}
}

func TestNewRouteMatcherSingle(t *testing.T) {
	// Single VirtualHost
	for key, vhConfig := range testVirutalHostConfigs {
		cfg := &v2.Proxy{
			VirtualHosts: []*v2.VirtualHost{vhConfig},
		}
		routers, err := NewRouteMatcher(cfg)
		if err != nil {
			t.Errorf("#%s : %v\n", key, err)
			continue
		}
		rm := routers.(*routeMatcher)
		expected := (rm.defaultVirtualHost != nil || len(rm.virtualHosts) == 1 || len(rm.wildcardVirtualHostSuffixes) == 1)
		if !expected {
			t.Errorf("#%s : expected one virtualhost\n")
		}
	}
}

func TestNewRouteMatcherGroup(t *testing.T) {
	//A group of VirtualHost
	var virtualhosts []*v2.VirtualHost
	for _, vhConfig := range testVirutalHostConfigs {
		virtualhosts = append(virtualhosts, vhConfig)
	}
	cfg := &v2.Proxy{
		VirtualHosts: virtualhosts,
	}
	routers, err := NewRouteMatcher(cfg)
	if err != nil {
		t.Error(err)
		return
	}
	rm := routers.(*routeMatcher)
	expected := (rm.defaultVirtualHost != nil && len(rm.virtualHosts) == 1 && len(rm.wildcardVirtualHostSuffixes) == 1)
	if !expected {
		t.Error("create routematcher not match")
	}
}

func TestNewRouteMatcherDuplicate(t *testing.T) {
	// two virtualhosts, both domain is "*", expected failed
	if _, err := NewRouteMatcher(&v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["all"], testVirutalHostConfigs["all"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	//two virtualhosts, both domain is "www.sofa-mosn.test", expected failed
	if _, err := NewRouteMatcher(&v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["domain"], testVirutalHostConfigs["domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with same suffix, expected failed
	if _, err := NewRouteMatcher(&v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{testVirutalHostConfigs["wildcard-domain"], testVirutalHostConfigs["wildcard-domain"]},
	}); err == nil {
		t.Error("expected an error occur, but not")
	}
	// wildcard domain with different suffix:
	// *.test.com, *.test.net, *.test.com.cn
	// expected OK
	if _, err := NewRouteMatcher(&v2.Proxy{
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Domains: []string{"*.test.com"}},
			&v2.VirtualHost{Domains: []string{"*.test.net"}},
			&v2.VirtualHost{Domains: []string{"*.test.com.cn"}},
		},
	}); err != nil {
		t.Error("NewRouteMatcher with different wildcard domain failed")
	}

}

func newTestSimpleRouter(name string) v2.Router {
	return v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{
				v2.HeaderMatcher{Name: "service", Value: ".*"},
			},
		},
		Route: v2.RouteAction{
			ClusterName: name,
		},
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
			unmatchedDomain: []string{"a.test.net", "a-test.com", ".test.com"},
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
		cfg := &v2.Proxy{
			VirtualHosts: []*v2.VirtualHost{vh},
		}
		routers, err := NewRouteMatcher(cfg)
		if err != nil {
			t.Errorf("#%d create routers failed: %v\n", i, err)
			continue
		}
		for _, match := range tc.matchedDomain {
			if routers.Route(map[string]string{
				strings.ToLower(protocol.MosnHeaderHostKey): match,
				"service": "test",
			}, 1) == nil {
				t.Errorf("%s expected matched: #%d, but return nil\n", match, i)
			}
		}
		for _, unmatch := range tc.unmatchedDomain {
			if routers.Route(map[string]string{
				strings.ToLower(protocol.MosnHeaderHostKey): unmatch,
				"service": "test",
			}, 1) != nil {
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
	cfg := &v2.Proxy{
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
		route := routers.Route(map[string]string{
			strings.ToLower(protocol.MosnHeaderHostKey): tc.Domain,
			"service": "test",
		}, 1)
		if route == nil {
			t.Errorf("%s match failed\n", tc.Domain)
			continue
		}
		if route.RouteRule().ClusterName() != tc.ExpectedRoute {
			t.Errorf("%s expected match %s, but got %s\n", tc.Domain, tc.ExpectedRoute, route.RouteRule().ClusterName())
		}
	}

}

func TestMain(m *testing.M) {
	log.InitDefaultLogger("", log.DEBUG)
	m.Run()
}
