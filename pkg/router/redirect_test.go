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
	"net/http"
	"reflect"
	"testing"

	"mosn.io/api"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestRedirectResponse(t *testing.T) {
	testCases := []struct {
		name           string
		redirectAction *v2.RedirectAction
		expectedRule   api.RedirectRule
		expectError    bool
	}{
		{
			name: "default response code is 301",
			redirectAction: &v2.RedirectAction{
				PathRedirect: "/bar",
			},
			expectedRule: &redirectImpl{
				code: http.StatusMovedPermanently,
				path: "/bar",
			},
		},
		{
			name: "explicitly set response code",
			redirectAction: &v2.RedirectAction{
				ResponseCode: http.StatusTemporaryRedirect,
				HostRedirect: "foo.com",
			},
			expectedRule: &redirectImpl{
				code: http.StatusTemporaryRedirect,
				host: "foo.com",
			},
		},
		{
			name: "unsupported response code",
			redirectAction: &v2.RedirectAction{
				ResponseCode: http.StatusOK,
				PathRedirect: "/bar",
			},
			expectError: true,
		},
		{
			name: "host and path redirect",
			redirectAction: &v2.RedirectAction{
				ResponseCode: http.StatusTemporaryRedirect,
				PathRedirect: "/foo",
				HostRedirect: "bar.com",
			},
			expectedRule: &redirectImpl{
				code: http.StatusTemporaryRedirect,
				path: "/foo",
				host: "bar.com",
			},
		},
		{
			name: "scheme redirect",
			redirectAction: &v2.RedirectAction{
				ResponseCode:   http.StatusTemporaryRedirect,
				PathRedirect:   "/foo",
				SchemeRedirect: "https",
			},
			expectedRule: &redirectImpl{
				code:   http.StatusTemporaryRedirect,
				path:   "/foo",
				scheme: "https",
			},
		},
	}

	match := v2.RouterMatch{
		Path: "/redirect",
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(testCase.name, func(t *testing.T) {
			routeCfg := &v2.Router{
				RouterConfig: v2.RouterConfig{
					Match:    match,
					Redirect: tc.redirectAction,
				},
			}
			rule, err := NewRouteRuleImplBase(nil, routeCfg)
			if tc.expectError {
				if err == nil {
					t.Fatalf("Unexpected success")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			rr := rule.RedirectRule()
			expectedRule := tc.expectedRule
			if rr == nil || reflect.ValueOf(rr).IsNil() {
				t.Fatal("RedirectResponseRule is nil")
			}
			if rr.RedirectScheme() != expectedRule.RedirectScheme() {
				t.Errorf("Unexpected scheme\nExpected: %s\nGot: %s", expectedRule.RedirectScheme(), rr.RedirectScheme())
			}
			if rr.RedirectHost() != expectedRule.RedirectHost() {
				t.Errorf("Unexpected host\nExpected: %s\nGot: %s", expectedRule.RedirectHost(), rr.RedirectHost())
			}
			if rr.RedirectPath() != expectedRule.RedirectPath() {
				t.Errorf("Unexpected path\nExpected: %s\nGot: %s", expectedRule.RedirectPath(), rr.RedirectPath())
			}
			if rr.RedirectCode() != expectedRule.RedirectCode() {
				t.Errorf("Unexpected response code\nExpected: %d\nGot: %d", expectedRule.RedirectCode(), rr.RedirectCode())
			}
		})
	}
}
