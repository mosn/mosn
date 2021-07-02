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

package faultinject

import (
	"context"

	"mosn.io/api"
)

// this file mocks the interface that used for test
// only implement the function that used in test
type mockStreamReceiverFilterCallbacks struct {
	api.StreamReceiverFilterHandler
	route      *mockRoute
	hijackCode int
	info       *mockRequestInfo
	called     chan int
}

func (cb *mockStreamReceiverFilterCallbacks) Route() api.Route {
	return cb.route
}
func (cb *mockStreamReceiverFilterCallbacks) RequestInfo() api.RequestInfo {
	return cb.info
}
func (cb *mockStreamReceiverFilterCallbacks) ContinueReceiving() {
	cb.called <- 1
}
func (cb *mockStreamReceiverFilterCallbacks) SendHijackReply(code int, headers api.HeaderMap) {
	cb.hijackCode = code
	cb.called <- 1
}

type mockRoute struct {
	api.Route
	rule *mockRouteRule
}

func (r *mockRoute) RouteRule() api.RouteRule {
	return r.rule
}

type mockRouteRule struct {
	api.RouteRule
	clustername string
	config      map[string]interface{}
}

func (r *mockRouteRule) ClusterName(context.Context) string {
	return r.clustername
}
func (r *mockRouteRule) PerFilterConfig() map[string]interface{} {
	return r.config
}

type mockRequestInfo struct {
	api.RequestInfo
	flag api.ResponseFlag
}

func (info *mockRequestInfo) SetResponseFlag(flag api.ResponseFlag) {
	info.flag = flag
}
