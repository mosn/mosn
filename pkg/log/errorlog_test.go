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
	"bufio"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"sofastack.io/sofa-mosn/pkg/types"
)

type errorLogCase struct {
	level Level
	f     func(format string, args ...interface{})
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

var preMapping = map[Level]string{
	FATAL: FatalPre,
	ERROR: ErrorPre,
	WARN:  WarnPre,
	INFO:  InfoPre,
	DEBUG: DebugPre,
	TRACE: TracePre,
}

func TestErrorLog(t *testing.T) {
	logName := "/tmp/mosn/error_log_print.log"
	os.Remove(logName)
	lg, err := GetOrCreateDefaultErrorLogger(logName, RAW)
	if err != nil {
		t.Fatal("create logger failed")
	}
	cases := []errorLogCase{
		{
			level: ERROR,
			f:     lg.Errorf,
		},
		{
			level: WARN,
			f:     lg.Warnf,
		},
		{
			level: INFO,
			f:     lg.Infof,
		},
		{
			level: DEBUG,
			f:     lg.Debugf,
		},
		{
			level: TRACE,
			f:     lg.Tracef,
		},
	}
	for _, c := range cases {
		lg.SetLogLevel(c.level)
		c.f("testdata")
	}
	lg.Toggle(true) // disable
	for _, c := range cases {
		lg.SetLogLevel(c.level)
		c.f("testdata") // write nothing
	}
	time.Sleep(time.Second) // wait buffer flush
	// read lines
	lines, err := readLines(logName)
	if err != nil {
		t.Fatal(err)
	}
	// verify count
	if len(lines) != len(cases) {
		t.Fatalf("logger write lines not expected, writes: %d, expected: %d", len(lines), len(cases))
	}
	// verify log in order if channel buffer is not full
	for i, l := range lines {
		// l format
		// 2006/01/02 15:04:05 [Level] {Count}
		qs := strings.Split(l, " ")
		c := cases[i]
		if !(len(qs) >= 4 && qs[2] == preMapping[c.level]) {
			t.Errorf("level: %v write format is not expected", c)
		}
	}
}

func TestAlertLog(t *testing.T) {
	logName := "/tmp/mosn/alert_test.log"
	os.Remove(logName)
	// common logger
	lg, err := GetOrCreateDefaultErrorLogger(logName, RAW)
	if err != nil {
		t.Fatal("create default logger failed")
	}
	// proxy logger
	plg, err := CreateDefaultProxyLogger(logName, RAW)
	if err != nil {
		t.Fatal("create proxy logger failed")
	}
	var testKey types.ErrorKey = types.ErrorModuleMosn + "test.alert"
	lg.Alertf(testKey, "test-alert-log")
	plg.Alertf(context.Background(), testKey, "test-%s", "alert-log")
	time.Sleep(time.Second) // wait log flush
	lines, err := readLines(logName)
	if err != nil {
		t.Fatal(err)
	}
	// {date time} [ERROR] [sofa-mosn.test.alert] test-alert-log
	// {date time} [ERROR] [sofa-mosn.test.alert] [-,-] test-alert-log
	if len(lines) < 2 {
		t.Fatal("not enough log output")
	}
	// spilt datetime
	defaultOut := strings.SplitN(lines[0], " ", 3)
	proxyOut := strings.SplitN(lines[1], " ", 3)
	if !(defaultOut[2] == "[ERROR] [sofa-mosn.test.alert] test-alert-log" &&
		proxyOut[2] == "[ERROR] [sofa-mosn.test.alert] [-,-] test-alert-log") {
		t.Errorf("log output is not expected, default output: %s , proxy output: %s", lines[0], lines[1])
	}

}

func BenchmarkLog(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := GetOrCreateDefaultErrorLogger("/tmp/mosn_bench/benchmark.log", DEBUG)
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		l.Debugf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
}

func BenchmarkLogParallel(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l, err := GetOrCreateDefaultErrorLogger("/tmp/mosn_bench/benchmark.log", DEBUG)
	if err != nil {
		b.Fatal(err)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Debugf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
		}
	})
}

func BenchmarkLogTimeNow(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		time.Now()
	}
}

func BenchmarkLogTimeFormat(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		time.Now().Format("2006/01/02 15:04:05.000")
	}
}

func BenchmarkLogTime(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		logTime()
	}
}
