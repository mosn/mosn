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
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/stats"
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
	stats.ResetAll()
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
			time.Sleep(300 * time.Duration(i) * time.Millisecond)
			tc := testCases[i]
			s := stats.NewStats(tc.typ, tc.namespace)
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
	//init prom
	flushInteval := time.Millisecond * 500
	sink := NewPromeSink(&PromConfig{
		Port:                  8088,
		Endpoint:              "/metrics",
		DisableCollectProcess: true,
		DisableCollectGo:      true,
	})

	times := 0
	tc := http.Client{}
	stopChan := make(chan bool)

	go func() {
		for {
			select {

			case <-time.Tick(flushInteval):
				sink.Flush(stats.GetAllRegistries())

				resp, _ := tc.Get("http://127.0.0.1:8088/metrics")
				ioutil.ReadAll(resp.Body)
				times++
			case <-stopChan:
				break
			}
		}
	}()

	wg.Wait()
	stopChan <- true
	typs := stats.LisTypes()
	if !(len(typs) == 2 &&
		typs[0] == "t1" &&
		typs[1] == "t2") {
		t.Error("types record error")
	}
}
