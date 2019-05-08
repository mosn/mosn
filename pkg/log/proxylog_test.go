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
	"github.com/alipay/sofa-mosn/pkg/types"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
)

func TestProxyLog(t *testing.T) {
	logName := "/tmp/mosn/proxy_log_print.log"
	os.Remove(logName)
	lg, err := CreateDefaultProxyLogger(logName, RAW)
	if err != nil {
		t.Fatal("create logger failed")
	}

	traceId := "0abfc19515355177863163255e6d87"
	connId := uint64(rand.Intn(10))
	targetStr := fmt.Sprintf("[%v,%v]", connId, traceId)
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyTraceId, traceId)
	ctx = mosnctx.WithValue(ctx, types.ContextKeyConnectionID, connId)

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
			t.Errorf("line %v write format is not expected", i)
		}
	}

}

func BenchmarkProxyLog(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := CreateDefaultProxyLogger("/tmp/mosn_bench/benchmark.log", DEBUG)
	if err != nil {
		b.Fatal(err)
	}

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyTraceId, "0abfc19515355177863163255e6d87")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		l.Infof(ctx, "BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
}

func BenchmarkProxyLogParallel(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := CreateDefaultProxyLogger("/tmp/mosn_bench/benchmark.log", DEBUG)
	if err != nil {
		b.Fatal(err)
	}

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyTraceId, "0abfc19515355177863163255e6d87")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Debugf(ctx, "BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
		}
	})
}
