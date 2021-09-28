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
	"testing"
	"time"

	"mosn.io/api"
)

func TestParseStreamFaultInjectFilter(t *testing.T) {
	m := map[string]interface{}{
		"delay": map[string]interface{}{
			"fixed_delay": "1s",
			"percentage":  100,
		},
		"abort": map[string]interface{}{
			"status":     500,
			"percentage": 100,
		},
		"upstream_cluster": "clustername",
		"headers": []interface{}{
			map[string]interface{}{
				"name":  "service",
				"value": "test",
				"regex": false,
			},
			map[string]interface{}{
				"name":  "user",
				"value": "bob",
				"regex": false,
			},
		},
	}
	faultInject, err := ParseStreamFaultInjectFilter(m)
	if err != nil {
		t.Error("parse stream fault inject failed")
		return
	}
	if !(faultInject.UpstreamCluster == "clustername" &&
		len(faultInject.Headers) == 2 &&
		faultInject.Abort != nil &&
		faultInject.Delay != nil) {
		t.Error("parse stream fault inject unexpected")
		return
	}
	if !(faultInject.Abort.Percent == 100 && faultInject.Abort.Status == 500) {
		t.Error("parse stream fault inject's abort unexpected")
	}
	if !(faultInject.Delay.Percent == 100 && faultInject.Delay.Delay == time.Second) {
		t.Error("parse stream fault inject's delay unexpected")
	}
}

type mockStreamFilterChainFactoryCallbacks struct {
	api.StreamFilterChainFactoryCallbacks
	rf api.StreamReceiverFilter
	p  api.ReceiverFilterPhase
}

func (m *mockStreamFilterChainFactoryCallbacks) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	m.rf = filter
	m.p = p
}

func TestFactory(t *testing.T) {
	fac, err := CreateFaultInjectFilterFactory(map[string]interface{}{})
	if err != nil {
		t.Fatalf("create factory failed: %v", err)
	}
	cb := &mockStreamFilterChainFactoryCallbacks{}
	fac.CreateFilterChain(context.TODO(), cb)
	if cb.rf == nil || cb.p != api.AfterRoute {
		t.Fatalf("create filter chain failed")
	}
}
