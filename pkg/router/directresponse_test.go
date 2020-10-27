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

	jsoniter "github.com/json-iterator/go"
	v2 "mosn.io/mosn/pkg/config/v2"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func TestDirectResponse(t *testing.T) {
	routeConfigStr := `{
		"match": {
			"prefix": "/"
		},
		"route": {
			"cluster_name":"testcluster"
		},
		"direct_response": {
			"status": 500,
			"body": "test"
		}
	}`
	routeCfg := &v2.Router{}
	if err := json.Unmarshal([]byte(routeConfigStr), routeCfg); err != nil {
		t.Fatal("unmarshal config to router failed, ", err)
	}
	rule, _ := NewRouteRuleImplBase(nil, routeCfg)
	if rule.DirectResponseRule() == nil {
		t.Fatal("rule have no direct response rule")
	}
	dr := rule.DirectResponseRule()
	if dr.StatusCode() != 500 || dr.Body() != "test" {
		t.Error("direct response rule is not exepcted")
	}
	// Test No Direct response by default
	noDirectCfgStr := `{
		"match": {
			"prefix": "/"
		},
		"route": {
			 "cluster_name":"testcluster"
		}
	}`
	noDirectCfg := &v2.Router{}
	if err := json.Unmarshal([]byte(noDirectCfgStr), noDirectCfg); err != nil {
		t.Fatal("unmarshal config to router failed, ", err)
	}
	noDirectRule, _ := NewRouteRuleImplBase(nil, noDirectCfg)
	if !reflect.ValueOf(noDirectRule.DirectResponseRule()).IsNil() {
		t.Error("expected a nil resposne rule, but not", noDirectRule.DirectResponseRule())
	}
}
