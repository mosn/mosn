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
	"fmt"
	"testing"

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
