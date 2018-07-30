package router

import (
	"regexp"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
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
			Match: v2.RouterMatch{Prefix: tc.prefix},
			Route: v2.RouteAction{ClusterName: "test"},
		}
		rr := &PrefixRouteRuleImpl{
			NewRouteRuleImplBase(virtualHostImpl, route),
			route.Match.Prefix,
		}
		headers := map[string]string{protocol.MosnHeaderPathKey: tc.headerpath}
		result := (rr.Match(headers, 1) != nil)
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
			Match: v2.RouterMatch{Path: tc.path},
			Route: v2.RouteAction{ClusterName: "test"},
		}
		base := NewRouteRuleImplBase(virtualHostImpl, route)
		base.caseSensitive = tc.caseSensitive //hack case sensitive
		rr := &PathRouteRuleImpl{base, route.Match.Path}
		headers := map[string]string{protocol.MosnHeaderPathKey: tc.headerpath}
		result := (rr.Match(headers, 1) != nil)
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
			Match: v2.RouterMatch{Regex: tc.regexp},
			Route: v2.RouteAction{ClusterName: "test"},
		}
		re := regexp.MustCompile(tc.regexp)
		rr := &RegexRouteRuleImpl{
			NewRouteRuleImplBase(virtualHostImpl, route),
			route.Match.Regex,
			*re,
		}
		headers := map[string]string{protocol.MosnHeaderPathKey: tc.headerpath}
		result := (rr.Match(headers, 1) != nil)
		if result != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
	}
}
