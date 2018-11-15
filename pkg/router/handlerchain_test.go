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
	headerMatch := protocol.CommonHeader(map[string]string{
		"test": "test",
	})
	routers := &mockRouters{
		r: []types.Route{
			&mockRouter{},
		},
		header: headerMatch,
	}
	//
	makeHandlerChain = DefaultMakeHandlerChain
	//
	if hc := CallMakeHandlerChain(headerMatch, routers); hc == nil {
		t.Fatal("make handler chain failed")
	} else {
		if r := hc.DoNextHandler(); r == nil {
			t.Fatal("do next handler failed")
		}
	}
	headerNotMatch := protocol.CommonHeader(map[string]string{})
	if hc := CallMakeHandlerChain(headerNotMatch, routers); hc != nil {
		t.Fatal("make handler chain unexpected")
	}

}
