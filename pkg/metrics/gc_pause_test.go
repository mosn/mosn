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

package metrics

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/types"
)

func gcPause() time.Duration {
	runtime.GC()
	var stats debug.GCStats
	debug.ReadGCStats(&stats)
	return stats.PauseTotal
}

const (
	entries = 200000
)

func TestGCPause(t *testing.T) {
	debug.SetGCPercent(10)
	fmt.Println("Number of entries: ", entries)

	r := gometrics.NewRegistry()
	for i := 0; i < entries; i++ {
		key := fmt.Sprintf("key-%010d", i)
		r.GetOrRegister(key, gometrics.NewGauge).(gometrics.Gauge).Update(int64(i))
	}
	key := fmt.Sprintf("key-%010d", 0)
	if r.Get(key).(gometrics.Gauge).Value() != 0 {
		t.Error("gometrics entry value not correct")
	}
	fmt.Println("GC pause for gometrics: ", gcPause())

	r = nil
	gcPause()

	//------------------------------------------

	// just for test
	originPath := types.MosnConfigPath
	types.MosnConfigPath = "."

	defer func() {
		types.MosnConfigPath = originPath
	}()
	zone := shm.InitMetricsZone("testGCPause", entries*256)
	defer func() {
		zone.Detach()
		shm.Reset()
	}()

	m, _ := NewMetrics("test", map[string]string{"type": "gc"})
	for i := 0; i < entries; i++ {
		key := fmt.Sprintf("key-%010d", i)
		m.Gauge(key).Update(int64(i))
	}

	key = fmt.Sprintf("key-%010d", 0)
	if m.Gauge(key).Value() != 0 {
		t.Error("shm-based entry value not correct")
	}

	fmt.Println("GC pause for shm-based metrics: ", gcPause())
	ResetAll()
}
