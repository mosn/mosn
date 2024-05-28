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

package log

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
	"mosn.io/pkg/variable"
)

func TestProxyLog(t *testing.T) {
	logName := "/tmp/mosn/proxy_log_print.log"
	os.Remove(logName)
	lg, err := CreateDefaultContextLogger(logName, log.RAW)
	if err != nil {
		t.Fatal("create logger failed")
	}

	traceId := "0abfc19515355177863163255e6d87"
	connId := uint64(rand.Intn(10))
	upstreamConnID := uint64(rand.Intn(10))
	targetStr := fmt.Sprintf("[%v,%v,%v]", connId, upstreamConnID, traceId)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableTraceId, traceId)
	_ = variable.Set(ctx, types.VariableConnectionID, connId)
	_ = variable.Set(ctx, types.VariableUpstreamConnectionID, upstreamConnID)

	for i := 0; i < 10; i++ {
		lg.Infof(ctx, "[unittest] test write, round %d", i)
	}

	time.Sleep(time.Second) // wait buffer flush
	// read lines
	lines, err := readLines(logName)

	// verify log in order if channel buffer is not full
	for i, l := range lines {
		// l format
		//  {time} [{level}] [{connId},{traceId}] {content}
		if strings.Index(l, targetStr) < 0 {
			t.Errorf("line %v write format is not expected: %s", i, l)
		}
	}
}

func TestProxyLog2(t *testing.T) {
	logName := "/tmp/mosn/proxy_log_print2.log"
	os.Remove(logName)
	lg, err := CreateDefaultContextLogger(logName, log.RAW)
	if err != nil {
		t.Fatal("create logger failed")
	}
	traceId := "0abfc19515355177863163255e6d87"
	connId := uint64(1)
	upstreamConnID := uint64(2)
	proxyMsg := fmt.Sprintf("[%d,%d,%s]", connId, upstreamConnID, traceId)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableTraceId, traceId)
	_ = variable.Set(ctx, types.VariableConnectionID, connId)
	_ = variable.Set(ctx, types.VariableUpstreamConnectionID, upstreamConnID)

	funcs := []func(ctx context.Context, format string, args ...interface{}){
		lg.Infof,
		lg.Warnf,
		lg.Debugf,
	}
	for i, f := range funcs {
		f(ctx, "test_%d", i)
	}
	// test error level
	lg.Errorf(ctx, "test_%s", "error")               // 2006-01-02 15:04:05,000 [ERROR] [connId, traceId] msg
	lg.Alertf(ctx, "mosn.alert", "test_%s", "alert") // 2006-01-02 15:04:05,000 [ERROR] [mosn.alert] [connId, traceId] msg
	time.Sleep(time.Second)                          // wait buffer flush
	// read lines
	lines, err := readLines(logName)
	for i, l := range lines[:len(funcs)] {
		qs := strings.Split(l, " ")
		// 2006-01-02 15:04:05,000 [LEVEL] [connId, traceId] msg
		if !(len(qs) == 5 &&
			qs[3] == proxyMsg &&
			qs[4] == fmt.Sprintf("test_%d", i)) {
			t.Fatalf("%d write unexpected data: %s", i, l)
		}
	}
	errMsgs := strings.Split(lines[3], " ")
	if !(len(errMsgs) == 5 &&
		errMsgs[3] == proxyMsg &&
		errMsgs[4] == "test_error") {
		t.Fatalf("error lines unexpected: %s", lines[3])
	}
	alertMsgs := strings.Split(lines[4], " ")
	if !(len(alertMsgs) == 6 &&
		alertMsgs[3] == "[mosn.alert]" &&
		alertMsgs[4] == proxyMsg &&
		alertMsgs[5] == "test_alert") {
		t.Fatalf("alert lines unexpected: %s", lines[4])
	}
}

func BenchmarkProxyLog(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := CreateDefaultContextLogger("/tmp/mosn_bench/benchmark.log", log.DEBUG)
	if err != nil {
		b.Fatal(err)
	}

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableTraceId, "0abfc19515355177863163255e6d87")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		l.Infof(ctx, "BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
}

func BenchmarkProxyLogParallel(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := CreateDefaultContextLogger("/tmp/mosn_bench/benchmark.log", log.DEBUG)
	if err != nil {
		b.Fatal(err)
	}

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableTraceId, "0abfc19515355177863163255e6d87")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Debugf(ctx, "BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
		}
	})
}
