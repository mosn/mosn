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

package stats

import (
	"sync"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
)

// testClear cleans the registry for test
func testClear() {
	reg = &registry{
		registries: make(map[string]metrics.Registry),
		mutex:      sync.RWMutex{},
	}
}

func TestTrimKey(t *testing.T) {
	testClear()
	typ := "test@type"
	namespace := "test@namespace"
	s := NewStats(typ, namespace)
	if !(s.typ == "testtype" && s.namespace == "testnamespace") {
		t.Error("type/namespace have at, it is not allowed")
	}
}

type testAction int

const (
	countInc testAction = iota
	countDec
	gaugeUpdate
	histogramUpdate
)

// test concurrently add statisic data
// should get the right data
func TestMetrics(t *testing.T) {
	testClear()
	testCases := []struct {
		typ         string
		namespace   string
		key         string
		action      testAction
		actionValue int64
	}{
		{"t1", "ns1", "k1", countInc, 1},
		{"t1", "ns1", "k1", countDec, 1},
		{"t1", "ns1", "k2", countInc, 1},
		{"t1", "ns1", "k3", gaugeUpdate, 1},
		{"t1", "ns1", "k4", histogramUpdate, 1},
		{"t1", "ns1", "k4", histogramUpdate, 2},
		{"t1", "ns1", "k4", histogramUpdate, 3},
		{"t1", "ns1", "k4", histogramUpdate, 4},
		{"t1", "ns2", "k1", countInc, 1},
		{"t1", "ns2", "k2", countInc, 2},
		{"t1", "ns2", "k3", gaugeUpdate, 3},
		{"t1", "ns2", "k4", histogramUpdate, 2},
		{"t2", "ns1", "k1", countInc, 1},
	}
	wg := sync.WaitGroup{}
	for i := range testCases {
		wg.Add(1)
		go func(i int) {
			tc := testCases[i]
			s := NewStats(tc.typ, tc.namespace)
			switch tc.action {
			case countInc:
				s.Counter(tc.key).Inc(tc.actionValue)
			case countDec:
				s.Counter(tc.key).Dec(tc.actionValue)
			case gaugeUpdate:
				s.Gauge(tc.key).Update(tc.actionValue)
			case histogramUpdate:
				s.Histogram(tc.key).Update(tc.actionValue)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	typs := LisTypes()
	if !(len(typs) == 2 &&
		typs[0] == "t1" &&
		typs[1] == "t2") {
		t.Error("types record error")
	}
	datas := GetAllMetricsData()
	t1Data := datas["t1"]
	if ns1, ok := t1Data["ns1"]; !ok {
		t.Error("no ns1 data")
	} else {
		if !(ns1["k1"] == "0" &&
			ns1["k2"] == "1" &&
			ns1["k3"] == "1") {
			t.Error("count and gauge not expected")
		}
		//TODO: histogram value expected
	}
	if ns2, ok := t1Data["ns2"]; !ok {
		t.Error("no ns2 data")
	} else {
		if !(ns2["k1"] == "1" &&
			ns2["k2"] == "2" &&
			ns2["k3"] == "3") {
			t.Error("count and gauge not expected")
		}
		//TODO: histogram value expected
	}
	t2Data := datas["t2"]
	if ns1, ok := t2Data["ns1"]; !ok {
		t.Error("no ns1 data")
	} else {
		if ns1["k1"] != "1" {
			t.Error("k1 value not expected")
		}
	}

}

func BenchmarkGetMetrics(b *testing.B) {
	testClear()
	// init metrics data
	testCases := []struct {
		typ       string
		namespace string
	}{
		{DownstreamType, "proxyname"},
		{DownstreamType, "listener1"},
		{DownstreamType, "listener2"},
		{UpstreamType, "cluster1"},
		{UpstreamType, "cluster2"},
		{UpstreamType, "cluster1.host1"},
		{UpstreamType, "cluster1.host2"},
		{UpstreamType, "cluster2.host1"},
		{UpstreamType, "cluster2.host2"},
	}
	for _, tc := range testCases {
		s := NewStats(tc.typ, tc.namespace)
		s.Counter("key1").Inc(100)
		s.Counter("key2").Inc(100)
		s.Gauge("key3").Update(100)
		for i := 0; i < 5; i++ {
			s.Histogram("key4").Update(1)
		}
	}
	for i := 0; i < b.N; i++ {
		GetAllMetricsData()
	}
}
