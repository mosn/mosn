package proxy

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type mockRouters struct {
	r      []types.Route
	header types.HeaderMap
}
type mockRouter struct {
	types.Route
}

func (routers *mockRouters) Route(headers types.HeaderMap, randomValue uint64) types.Route {
	if reflect.DeepEqual(headers, routers.header) {
		return routers.r[0]
	}
	return nil
}
func (routers *mockRouters) GetAllRoutes(headers types.HeaderMap, randomValue uint64) []types.Route {
	if reflect.DeepEqual(headers, routers.header) {
		return routers.r
	}
	return nil
}

func TestDefaultMakeHandlerChain(t *testing.T) {
	header_match := protocol.CommonHeader(map[string]string{
		"test": "test",
	})
	routers := &mockRouters{
		r: []types.Route{
			&mockRouter{},
		},
		header: header_match,
	}
	if hc := DefaultMakeHandlerChain(header_match, routers); hc == nil {
		t.Fatal("make handler chain failed")
	} else {
		if r := hc.DoNextHandler(); r == nil {
			t.Fatal("do next handler failed")
		}
	}
	header_notmatch := protocol.CommonHeader(map[string]string{})
	if hc := DefaultMakeHandlerChain(header_notmatch, routers); hc != nil {
		t.Fatal("make handler chain unexpected")
	}

}
