package router

import (
	"github.com/alipay/sofa-mosn/internal/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"testing"
)

type testCase struct {
}

func TestPrefixRouteRuleImpl_Match(t *testing.T) {
	testCase := []string{
		"/", "/test", "/test_", "/test_prefix",
	}

	invalidTestCase := []string{
		"./", "./test",
	}

	examplePrefix := "/"

	var router types.Routers
	var err error

	if router, err = newPrefixVirtualHost(examplePrefix); err != nil {
		t.Error(err)
	}

	for _, testcase := range testCase {
		if r := router.Route(map[string]string{protocol.MosnHeaderPathKey: testcase}, 1); r == nil {
			t.Errorf("want match, but got nil")
		} else {
			t.Log("clustername:", r.RouteRule().ClusterName())
		}
	}

	for _, testcase := range invalidTestCase {
		if r := router.Route(map[string]string{protocol.MosnHeaderPathKey: testcase}, 1); r != nil {
			t.Errorf("want no match, but got one")
		}
	}

}

func newPrefixVirtualHost(prefix string) (types.Routers, error) {
	virtualHosts := []*v2.VirtualHost{
		&v2.VirtualHost{Domains: []string{"*"}, Routers: []v2.Router{newPrefixRouter(prefix)}},
	}
	cfg := &v2.Proxy{
		VirtualHosts: virtualHosts,
	}

	return NewRouteMatcher(cfg)
}

func newPrefixRouter(prefix string) v2.Router {
	return v2.Router{
		Match: v2.RouterMatch{
			Prefix: prefix,
		},
		Route: v2.RouteAction{
			ClusterName: prefix,
		},
	}
}
