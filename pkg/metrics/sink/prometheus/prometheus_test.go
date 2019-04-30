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

package prometheus

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	"time"

	"fmt"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/alipay/sofa-mosn/pkg/metrics/sink"
)

type testAction int

const (
	countInc testAction = iota
	countDec
	gaugeUpdate
	histogramUpdate
)

// test concurrently add statisic data
// should get the right data from prometheus
func TestPrometheusMetrics(t *testing.T) {
	//zone := shm.InitMetricsZone("TestPrometheusMetrics", 10*1024)
	//defer zone.Detach()

	metrics.ResetAll()
	testCases := []struct {
		typ         string
		labels      map[string]string
		key         string
		action      testAction
		actionValue int64
	}{
		{"t1", map[string]string{"lbk1": "lbv1"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv2"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k1", countDec, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k2", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k3", gaugeUpdate, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 2},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 3},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 4},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k2", countInc, 2},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k3", gaugeUpdate, 3},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k4", histogramUpdate, 2},
		{"t2", map[string]string{"lbk1": "lbv1"}, "k1", countInc, 1},
	}
	wg := sync.WaitGroup{}
	for i := range testCases {
		wg.Add(1)
		go func(i int) {
			tc := testCases[i]
			s, _ := metrics.NewMetrics(tc.typ, tc.labels)
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

	sink := NewPromeSink(&promConfig{
		Port:     8088,
		Endpoint: "/metrics",
		//DisableCollectProcess: true,
		//DisableCollectGo:      true,
	})
	_ = sink

	store.StartService(nil)
	defer store.StopService()
	time.Sleep(time.Second) // wait server start

	tc := http.Client{}

	resp, err := tc.Get("http://127.0.0.1:8088/metrics")
	if err != nil {
		// wait listener ready
		time.Sleep(time.Second)
		resp, err = tc.Get("http://127.0.0.1:8088/metrics")

		// still error
		if err != nil {
			t.Error("get metrics failed:", err)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if !bytes.Contains(body, []byte("lbk1_t1_k1{lbk1=\"lbv1\"} 0.0")) {
		t.Error("lbk1_t1_k1{lbk1=\"lbv1\"} metric not correct")
	}

	if !bytes.Contains(body, []byte("lbk1_t1_k1{lbk1=\"lbv2\"} 1.0")) {
		t.Error("lbk1_t1_k1{lbk1=\"lbv2\"} metric not correct")
	}

	if !bytes.Contains(body, []byte("lbk1_t1_k4_max{lbk1=\"lbv1\"} 4.0")) {
		t.Error("lbk1_t1_k4_max{lbk1=\"lbv1\"} metric not correct")
	}

	if !bytes.Contains(body, []byte("lbk2_t1_k4_min{lbk2=\"lbv2\"} 2.0")) {
		t.Error("lbk2_t1_k4_min{lbk2=\"lbv2\"} metric not correct")
	}
}

func TestPrometheusMetricsFilter(t *testing.T) {
	metrics.ResetAll()
	testCases := []struct {
		typ         string
		labels      map[string]string
		key         string
		action      testAction
		actionValue int64
	}{
		{"t1", map[string]string{"lbk1": "lbv1"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv2"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k1", countDec, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k2", countInc, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k3", gaugeUpdate, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 1},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 2},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 3},
		{"t1", map[string]string{"lbk1": "lbv1"}, "k4", histogramUpdate, 4},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k1", countInc, 1},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k2", countInc, 2},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k3", gaugeUpdate, 3},
		{"t1", map[string]string{"lbk2": "lbv2"}, "k4", histogramUpdate, 2},
		{"t2", map[string]string{"lbk1": "lbv1"}, "k1", countInc, 1},
	}
	wg := sync.WaitGroup{}
	for i := range testCases {
		wg.Add(1)
		go func(i int) {
			tc := testCases[i]
			s, _ := metrics.NewMetrics(tc.typ, tc.labels)
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

	NewPromeSink(&promConfig{
		Port:     8088,
		Endpoint: "/metrics",
		//DisableCollectProcess: true,
		//DisableCollectGo:      true,
	})

	store.StartService(nil)
	defer store.StopService()
	time.Sleep(time.Second) // wait server start

	tc := http.Client{}

	// SetFilter
	sink.SetFilterLabels([]string{"lbk1"})
	sink.SetFilterKeys([]string{"k1"})

	resp, err := tc.Get("http://127.0.0.1:8088/metrics")
	if err != nil {
		// wait listener ready
		time.Sleep(time.Second)
		resp, err = tc.Get("http://127.0.0.1:8088/metrics")

		// still error
		if err != nil {
			t.Error("get metrics failed:", err)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if bytes.Contains(body, []byte("lbk1")) {
		t.Error("filter set label: lbk1, but still flush")
	}
	if bytes.Contains(body, []byte("k1")) {
		t.Error("filter set key: k1 , but still flush")
	}

	if !bytes.Contains(body, []byte("lbk2_t1_k4_min{lbk2=\"lbv2\"} 2.0")) {
		t.Error("lbk2_t1_k4_min{lbk2=\"lbv2\"} metric not correct")
	}
}

func TestPrometheusFlatternKey(t *testing.T) {
	testcase := []struct {
		input  string
		output string
	}{
		{
			input:  "listener.127.0.0.1:34903",
			output: "listener_127_0_0_1:34903",
		},
		{
			input:  "go_version:go1.9",
			output: "go_version:go1_9",
		},
		{
			input:  "listener_address:0.0.0.0:9529",
			output: "listener_address:0_0_0_0:9529",
		},
	}

	for _, c := range testcase {
		if flattenKey(c.input) != c.output {
			t.Error("prometheus flattern key error:", c)
		}
	}
}

func BenchmarkPromSink_Flush(b *testing.B) {
	//zone := shm.InitMetricsZone("BenchmarkPromSink_Flush", 50*1024*1024)
	//defer zone.Detach()

	// 5000 registry + each registry 40 metrics
	for i := 0; i < 5000; i++ {
		m, _ := metrics.NewMetrics(fmt.Sprintf("type%d", i), map[string]string{
			fmt.Sprintf("lbk%d", i): fmt.Sprintf("lbv%d", i),
		})

		for j := 0; j < 40; j++ {
			m.Gauge(fmt.Sprintf("gg%d", j))
		}

	}

	sink := NewPromeSink(&promConfig{
		Port:                  8088,
		Endpoint:              "/metrics",
		DisableCollectProcess: true,
		DisableCollectGo:      true,
		//DisablePassiveFlush:   true,
	})
	_ = sink
	store.StartService(nil)
	defer store.StopService()
	time.Sleep(time.Second) // wait server start

	//tc := http.Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink.Flush(ioutil.Discard, metrics.GetAll())
		//url, _ := url.Parse("http://127.0.0.1:8088/metrics")
		//tc.Do(&http.Request{
		//	Method: http.MethodGet,
		//	URL:    url,
		//	//Header: map[string][]string{
		//	//	//"Accept-Encoding": {"gzip, deflate"},
		//	//	"Accept": {"application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily encoding=compact-text"},
		//	//},
		//})
		//resp, err := tc.Get("http://127.0.0.1:8088/metrics")
		//if err != nil {
		//	b.Error("get metrics failed:", err)
		//}
		//io.Copy(ioutil.Discard, resp.Body)
	}
}

func BenchmarkPromSink_Filter(b *testing.B) {
	// 5000 registry + each registry 40 metrics
	for i := 0; i < 5000; i++ {
		m, _ := metrics.NewMetrics(fmt.Sprintf("type%d", i), map[string]string{
			fmt.Sprintf("lbk%d", i): fmt.Sprintf("lbv%d", i),
		})

		for j := 0; j < 40; j++ {
			m.Gauge(fmt.Sprintf("gg%d", j))
		}

	}

	psink := NewPromeSink(&promConfig{
		Port:                  8088,
		Endpoint:              "/metrics",
		DisableCollectProcess: true,
		DisableCollectGo:      true,
	})
	// without filter
	b.Run("common flush", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			psink.Flush(ioutil.Discard, metrics.GetAll())
		}
	})
	// filter all registry
	filterLabels := make([]string, 0, 5000)
	for i := 0; i < 5000; i++ {
		filterLabels = append(filterLabels, fmt.Sprintf("lbk%d", i))
	}
	sink.SetFilterLabels(filterLabels)
	b.Run("filter flush", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			psink.Flush(ioutil.Discard, metrics.GetAll())
		}
	})

}
