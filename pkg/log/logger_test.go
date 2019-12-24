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
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	"sofastack.io/sofa-mosn/pkg/buffer"
)

func TestLogPrintDiscard(t *testing.T) {
	l, err := GetOrCreateLogger("/tmp/mosn_bench/benchmark.log", nil)
	if err != nil {
		t.Fatal(err)
	}
	buf := buffer.GetIoBuffer(100)
	buf.WriteString("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog")
	l.Close()
	runtime.Gosched()
	// writeBufferChan is 1000
	// l.Printf is discard, non block
	for i := 0; i < 1001; i++ {
		l.Printf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
	lchan := make(chan struct{})
	go func() {
		// block
		l.Print(buf, false)
		lchan <- struct{}{}
	}()

	select {
	case <-lchan:
		t.Errorf("test Print diacard failed, should be block")
	case <-time.After(time.Second * 3):
	}
}

func TestLogPrintnull(t *testing.T) {
	logName := "/tmp/mosn_bench/printnull.log"
	os.Remove(logName)
	l, err := GetOrCreateLogger(logName, nil)
	if err != nil {
		t.Fatal(err)
	}
	buf := buffer.GetIoBuffer(0)
	buf.WriteString("testlog")
	l.Print(buf, false)
	buf = buffer.GetIoBuffer(0)
	buf.WriteString("")
	l.Print(buf, false)
	l.Close()
	time.Sleep(time.Second)
	f, _ := os.Open(logName)
	b := make([]byte, 1024)
	n, _ := f.Read(b)
	f.Close()

	if n != len("testlog") {
		t.Errorf("Printnull error")
	}
	if string(b[:n]) != "testlog" {
		t.Errorf("Printnull error")
	}
}

func TestLogDefaultRollerTime(t *testing.T) {
	logName := "/tmp/mosn_bench/printdefaultroller.log"
	rollerName := logName + "." + time.Now().Format("2006-01-02_15")
	os.Remove(logName)
	os.Remove(rollerName)
	// 2s
	logger, err := GetOrCreateLogger(logName, &Roller{MaxTime: 2})
	if err != nil {
		t.Fatal(err)
	}
	logger.Print(buffer.NewIoBufferString("1111111"), false)
	time.Sleep(3 * time.Second)
	logger.Print(buffer.NewIoBufferString("2222222"), false)
	time.Sleep(1 * time.Second)
	logger.Close()

	b := make([]byte, 100)
	f, err := os.Open(logName)
	if err != nil {
		t.Errorf("TestLogDefaultRoller failed %v", err)
	}
	n, err := f.Read(b)
	f.Close()
	if err != io.EOF || n != 0 {
		t.Errorf("TestLogDefaultRoller failed %v", err)
	}

	f, err = os.Open(rollerName)
	if err != nil {
		t.Errorf("TestLogDefaultRoller failed %v", err)
	}
	n, err = f.Read(b)
	f.Close()
	if n == 0 || string(b[0:n]) != "11111112222222" {
		t.Errorf("TestLogDefaultRoller failed %v", string(b[0:n]))
	}
}

func TestLogDefaultRollerAfterDelete(t *testing.T) {
	logName := "/tmp/log_roller_delete.log"
	rollerName := logName + "." + time.Now().Format("2006-01-02")
	os.Remove(logName)
	os.Remove(rollerName)

	logger, err := GetOrCreateLogger(logName, &Roller{MaxTime: 3})
	if err != nil {
		t.Fatal(err)
	}
	// remove the log file, the log is output to no where
	os.Remove(logName)
	logger.Print(buffer.NewIoBufferString("nowhere"), false)
	// wait roller
	time.Sleep(4 * time.Second)
	// the first log trigger roller will be writed in the old file
	logger.Print(buffer.NewIoBufferString("ignore"), false)
	time.Sleep(100 * time.Millisecond) // wait write flush
	logger.Print(buffer.NewIoBufferString("output"), false)
	logger.Close() // force flush
	b, err := ioutil.ReadFile(logName)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if string(b) != "output" {
		t.Errorf("read file data: %s", string(b))
	}
	// rollerName should not be exists
	if _, err := os.Stat(rollerName); err == nil {
		t.Errorf("roller file exists, but expected not: %v", err)
	}
}

func TestLogReopen(t *testing.T) {
	l, err := GetOrCreateLogger("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.reopen(); err != ErrReopenUnsupported {
		t.Errorf("test log reopen failed")
	}
	l, err = GetOrCreateLogger("/tmp/mosn_bench/testlogreopen.log", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.reopen(); err != nil {
		t.Errorf("test log reopen failed %v", err)
	}
}

func TestLoglocalOffset(t *testing.T) {
	_, offset := time.Now().Zone()
	var defaultRollerTime int64 = 24 * 60 * 60
	t1 := time.Date(2018, time.December, 25, 23, 59, 59, 0, time.Local)
	t2 := time.Date(2018, time.December, 26, 00, 00, 01, 0, time.Local)
	if (t1.Unix()+int64(offset))/defaultRollerTime+1 != (t2.Unix()+int64(offset))/defaultRollerTime {
		t.Errorf("test localOffset failed")
	}
	t.Logf("t1=%d t2=%d offset=%d rollertime=%d\n", t1.Unix(), t2.Unix(), offset, defaultRollerTime)
	t.Logf("%d %d\n", (t1.Unix())/defaultRollerTime, (t1.Unix() / defaultRollerTime))
	t.Logf("%d %d\n", (t1.Unix()+int64(offset))/defaultRollerTime, (t2.Unix()+int64(offset))/defaultRollerTime)
}
