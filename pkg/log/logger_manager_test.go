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
	"os"
	"strings"
	"testing"
	"time"

	"mosn.io/pkg/log"
)

func TestUpdateLoggerConfig(t *testing.T) {
	// reset for test
	errorLoggerManagerInstance.managers = make(map[string]log.ErrorLogger)
	log.ClearAll()
	//
	logName := "/tmp/mosn/test_update_logger.log"
	if lg, err := GetOrCreateDefaultErrorLogger(logName, log.DEBUG); err != nil {
		t.Fatal(err)
	} else {
		if lg.(*errorLogger).Level != log.DEBUG {
			t.Fatal("logger created, but level is not expected")
		}
	}
	if lg, err := GetOrCreateDefaultErrorLogger(logName, log.INFO); err != nil {
		t.Fatal(err)
	} else {
		if lg.(*errorLogger).Level != log.DEBUG {
			t.Fatal("expected get a logger, not create a new one")
		}
	}
	// keeps the logger
	lg, _ := GetOrCreateDefaultErrorLogger(logName, log.RAW)
	logger := lg.(*errorLogger)
	if ok := UpdateErrorLoggerLevel("not_exists", log.INFO); ok {
		t.Fatal("update a not exists logger, expected failed")
	}
	// update log level, effects the logger
	if ok := UpdateErrorLoggerLevel(logName, log.TRACE); !ok {
		t.Fatal("update logger failed")
	} else {
		if logger.Level != log.TRACE {
			t.Fatal("update logger failed")
		}
	}
	// test disable/ enable
	if ok := ToggleLogger(logName, true); !ok {
		t.Fatal("disable logger failed")
	} else {
		if !logger.Logger.Disable() {
			t.Fatal("disbale logger failed")
		}
	}
	if ok := ToggleLogger(logName, false); !ok {
		t.Fatal("enable logger failed")
	} else {
		if logger.Logger.Disable() {
			t.Fatal("enable logger failed")
		}
	}
	// Toggle Logger (not error logger)
	baseLoggerPath := "/tmp/mosn/base_logger.log"
	baseLogger, err := log.GetOrCreateLogger(baseLoggerPath, nil)
	if err != nil || baseLogger.Disable() {
		t.Fatalf("Create Logger not expected, error: %v, logger state: %v", err, baseLogger.Disable())
	}
	if ok := ToggleLogger(baseLoggerPath, true); !ok {
		t.Fatal("enable base logger failed")
	}
	if !baseLogger.Disable() {
		t.Fatal("disable Logger failed")
	}

}

func TestSetAllErrorLogLevel(t *testing.T) {
	defer log.CloseAll()
	// reset for test
	errorLoggerManagerInstance.managers = make(map[string]log.ErrorLogger)
	log.ClearAll()
	var logs []log.ErrorLogger
	for i := 0; i < 100; i++ {
		logName := fmt.Sprintf("/tmp/errorlog.%d.log", i)
		lg, err := GetOrCreateDefaultErrorLogger(logName, log.INFO)
		if err != nil {
			t.Fatal(err)
		}
		logs = append(logs, lg)
	}
	GetErrorLoggerManagerInstance().SetAllErrorLoggerLevel(log.ERROR)
	// verify
	for _, lg := range logs {
		if lg.GetLogLevel() != log.ERROR {
			t.Fatal("some error log's level is not changed")
		}
	}
}

func TestDefaultLoggerInit(t *testing.T) {
	logName := "/tmp/mosn/test_update_logger.log"
	// reset for test
	errorLoggerManagerInstance.managers = make(map[string]log.ErrorLogger)
	log.ClearAll()
	InitDefaultLogger(logName, ERROR)
	// Test Default Logger Level Update
	if !(DefaultLogger.GetLogLevel() == ERROR &&
		Proxy.GetLogLevel() == ERROR) {
		t.Fatal("init log failed")
	}
	UpdateErrorLoggerLevel(logName, INFO)
	if !(DefaultLogger.GetLogLevel() == INFO &&
		Proxy.GetLogLevel() == INFO) {
		t.Fatal("init log failed")
	}
}

// The error logger creator can be replaced to another implementation.
// The logger manager & proxy logger will use the same implementation.
func TestExtendErrorLogger(t *testing.T) {
	DefaultCreateErrorLoggerFunc = NewMockLogger
	defer func() {
		DefaultCreateErrorLoggerFunc = CreateDefaultErrorLogger
	}()
	logName := "/tmp/mosn/test_mock_log.log"
	os.Remove(logName)
	// reset for test
	errorLoggerManagerInstance.managers = make(map[string]log.ErrorLogger)
	log.ClearAll()
	if err := InitDefaultLogger(logName, INFO); err != nil {
		t.Fatal(err)
	}
	DefaultLogger.Infof("test_%d", 123)               // [mocked] [INFO] [] test_123
	Proxy.Infof(context.Background(), "test_%d", 123) // [mocked] [INFO] [] [connId,traceId] test_123
	time.Sleep(time.Second)
	lines, err := readLines(logName)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("logger write lines not expected, writes: %d, expected: %d", len(lines), 2)
	}
	for _, l := range lines {
		qs := strings.SplitN(l, " ", 4)
		if !(len(qs) == 4 &&
			qs[0] == "[mocked]" &&
			qs[1] == "[INFO]" &&
			qs[2] == "[]" &&
			strings.Contains(qs[3], "test_123")) {
			t.Fatalf("log output is unexpected: %s", l)
		}
	}
}

// mock a logger
type mockLogger struct {
	*log.SimpleErrorLog
}

func NewMockLogger(output string, level log.Level) (log.ErrorLogger, error) {
	lg, err := log.GetOrCreateLogger(output, nil)
	if err != nil {
		return nil, err
	}
	return &mockLogger{
		&log.SimpleErrorLog{
			Logger: lg,
			Level:  level,
			Formatter: func(lv string, alert string, format string) string {
				return fmt.Sprintf("[mocked] %s [%s] %s", lv, alert, format)
			},
		},
	}, nil
}
