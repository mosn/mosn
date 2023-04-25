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
	"testing"

	"github.com/golang/mock/gomock"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/types"
)

func TestRPCRouteRuleSimple(t *testing.T) {
	route := &v2.Router{}
	vh := &VirtualHostImpl{}
	base, err := NewRouteRuleImplBase(vh, route)
	if err != nil {
		t.Fatalf("create base route failed: %v", err)
	}
	headers := []v2.HeaderMatcher{
		{
			Name:  types.RPCRouteMatchKey,
			Value: ".*",
		},
	}
	rpcroute := CreateRPCRule(base, headers)
	if rpcroute.RouteRule().PathMatchCriterion().Matcher() != ".*" {
		t.Fatalf("sofa route rule should be fast match mode")
	}
	ctrl := gomock.NewController(t)
	mheader := mock.NewMockHeaderMap(ctrl)
	mheader.EXPECT().Get(types.RPCRouteMatchKey).Return("any value", true).AnyTimes()
	ctx := context.Background()
	if rpcroute.Match(ctx, mheader) == nil {
		t.Fatalf("sofa route rule  fast match failed")
	}
}

func TestSofaRouteRuleHeaderMatch(t *testing.T) {
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Headers: []v2.HeaderMatcher{
					{
						Name:  "key1",
						Value: "value1",
					},
					{
						Name:  "key2",
						Value: "value2",
					},
				},
			},
		},
	}
	vh := &VirtualHostImpl{}
	base, err := NewRouteRuleImplBase(vh, route)
	if err != nil {
		t.Fatalf("create base route failed: %v", err)
	}
	rpcroute := CreateRPCRule(base, route.Match.Headers)
	if rpcroute.RouteRule().PathMatchCriterion().Matcher() != "" {
		t.Fatalf("sofa route rule should not be fast match mode")
	}
	ctrl := gomock.NewController(t)
	// header matched
	headerMatched := mock.NewMockHeaderMap(ctrl)
	headerMatched.EXPECT().Get("key1").Return("value1", true).AnyTimes()
	headerMatched.EXPECT().Get("key2").Return("value2", true).AnyTimes()
	ctx := context.Background()
	if rpcroute.Match(ctx, headerMatched) == nil {
		t.Fatalf("sofa route rule matched header failed")
	}
	// header matched failed
	headerNotMatched := mock.NewMockHeaderMap(ctrl)
	headerNotMatched.EXPECT().Get("key1").Return("value1", true).AnyTimes()
	headerNotMatched.EXPECT().Get("key2").Return("", false).AnyTimes()
	if rpcroute.Match(ctx, headerNotMatched) != nil {
		t.Fatalf("sofa route rule matched success, but expected not")
	}

}
