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
	status types.HandlerStatus
}

func (r *mockRouter) RouteRule() types.RouteRule {
	return &mockRouteRule{}
}

type mockRouteRule struct {
	types.RouteRule
}

func (r *mockRouteRule) ClusterName() string {
	return ""
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

type mockManager struct {
	types.ClusterManager
}

func (m *mockManager) GetClusterSnapshot(ctx context.Context, name string) types.ClusterSnapshot {
	return nil
}
func (m *mockManager) PutClusterSnapshot(snapshot types.ClusterSnapshot) {
}

func resetHandlerChain() {
	makeHandlerChainOrder.makeHandlerChain = DefaultMakeHandlerChain
	makeHandlerChainOrder.order = 1
}

func TestDefaultMakeHandlerChain(t *testing.T) {
	headerMatch := protocol.CommonHeader(map[string]string{
		"test": "test",
	})
	routers := &mockRouters{
		r: []types.Route{
			&mockRouter{},
		},
		header: headerMatch,
	}
	// test register
	RegisterMakeHandlerChain(DefaultMakeHandlerChain, 10) // Register success
	RegisterMakeHandlerChain(_TestMakeHandlerChain, 1)    // Register faile
	defer resetHandlerChain()
	clusterManager := &mockManager{}
	// router match, handler available
	if hc := CallMakeHandlerChain(headerMatch, routers, clusterManager); hc == nil {
		t.Fatal("make handler chain failed")
	} else {
		if _, r := hc.DoNextHandler(); r == nil {
			t.Fatal("do next handler failed")
		}
	}
	// header not match, no handlers
	headerNotMatch := protocol.CommonHeader(map[string]string{})
	if hc := CallMakeHandlerChain(headerNotMatch, routers, clusterManager); hc == nil {
		t.Fatal("make handler chain unexpected")
	} else {
		if _, r := hc.DoNextHandler(); r != nil {
			t.Fatal("do next handler failed")
		}
	}

}

type mockStatusHandler struct {
	status types.HandlerStatus
	router types.Route
}

func (h *mockStatusHandler) IsAvailable(ctx context.Context, snapshot types.ClusterSnapshot) types.HandlerStatus {
	return h.status
}
func (h *mockStatusHandler) Route() types.Route {
	return h.router
}

func _TestMakeHandlerChain(headers types.HeaderMap, routers types.Routers, clusterManager types.ClusterManager) *RouteHandlerChain {
	rs := routers.GetAllRoutes(headers, 1)
	var handlers []types.RouteHandler
	for _, r := range rs {
		mockr := r.(*mockRouter)
		handler := &mockStatusHandler{
			status: mockr.status,
			router: r,
		}
		handlers = append(handlers, handler)
	}
	return NewRouteHandlerChain(context.Background(), clusterManager, handlers)
}

func TestExtendHandler(t *testing.T) {
	headerMatch := protocol.CommonHeader(map[string]string{
		"test": "test",
	})
	// Test HandlerChain: 1. NotAvailable 2. Stop
	routers := &mockRouters{
		r: []types.Route{
			&mockRouter{status: types.HandlerNotAvailable},
			&mockRouter{status: types.HandlerStop},
		},
		header: headerMatch,
	}
	// test register
	RegisterMakeHandlerChain(_TestMakeHandlerChain, 10)  // Register success
	RegisterMakeHandlerChain(DefaultMakeHandlerChain, 1) // Register failed
	defer resetHandlerChain()
	clusterManager := &mockManager{}
	//1.
	if hc := CallMakeHandlerChain(headerMatch, routers, clusterManager); hc == nil {
		t.Fatal("make extend handler chain failed")
	} else {
		if _, route := hc.DoNextHandler(); route != nil {
			t.Fatal("unexpected Handler result")
		}
	}
	// Test HandlerChain: 1. NotAvailable 2. Unexpected(as NotAvailable) 3. Available (checked)
	routers2 := &mockRouters{
		r: []types.Route{
			&mockRouter{status: types.HandlerNotAvailable},
			&mockRouter{status: types.HandlerStatus(-1)}, // Unexpected
			&mockRouter{},                                //Available
		},
		header: headerMatch,
	}
	if hc := CallMakeHandlerChain(headerMatch, routers2, clusterManager); hc == nil {
		t.Fatal("make extend handler chain failed")
	} else {
		if _, route := hc.DoNextHandler(); route == nil {
			t.Fatal("want to get a available router")
		} else {
			// verify the router
			if route.(*mockRouter).status != types.HandlerAvailable {
				t.Error("handler chain get router unexpected")
			}
		}
	}
	// Test HandlerChain: all of the handlers are NotAvailable
	routers3 := &mockRouters{
		r: []types.Route{
			&mockRouter{status: types.HandlerNotAvailable},
			&mockRouter{status: types.HandlerNotAvailable},
			&mockRouter{status: types.HandlerNotAvailable},
		},
		header: headerMatch,
	}
	if hc := CallMakeHandlerChain(headerMatch, routers3, clusterManager); hc == nil {
		t.Fatal("make extend handler chain failed")
	} else {
		if _, route := hc.DoNextHandler(); route != nil {
			t.Fatal("unexpected Handler result")
		}
	}
}
