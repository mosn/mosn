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
	"errors"
	"sync"

	"mosn.io/pkg/log"
)

var (
	DefaultLogger log.ErrorLogger
	StartLogger   log.ErrorLogger
	Proxy         log.ContextLogger

	ErrNoLoggerFound = errors.New("no logger found in logger manager")
)

var levelMap = map[log.Level]string{
	log.FATAL: "FATAL",
	log.ERROR: "ERROR",
	log.WARN:  "WARN",
	log.INFO:  "INFO",
	log.DEBUG: "DEBUG",
	log.TRACE: "TRACE",
}

var errorLoggerManagerInstance *ErrorLoggerManager

func init() {
	errorLoggerManagerInstance = &ErrorLoggerManager{
		mutex:    sync.Mutex{},
		managers: make(map[string]log.ErrorLogger),
	}
	// use console as start logger
	StartLogger, _ = GetOrCreateDefaultErrorLogger("", log.INFO)
	// default as start before Init
	log.DefaultLogger = StartLogger
	DefaultLogger = log.DefaultLogger
	// default proxy logger for test, override after config parsed
	log.DefaultContextLogger, _ = CreateDefaultContextLogger("", log.INFO)
	Proxy = log.DefaultContextLogger
}

// ErrorLoggerManager manages error log can be updated dynamicly
type ErrorLoggerManager struct {
	mutex    sync.Mutex
	disabled bool
	// if logLevelControl is set, it's the lowest log level over MOSN
	// for example: WARN level will limit DEBUG level to be WARN, when creating logger
	withLogLevelControl bool
	logLevelControl     log.Level
	managers            map[string]log.ErrorLogger
}

//GetAllErrorLogger returns all of ErrorLogger info
func (mng *ErrorLoggerManager) GetAllErrorLogger() map[string]string {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	loggers := make(map[string]string)
	log.DefaultLogger.Infof("logger is %+v", mng.managers)
	for key, lg := range mng.managers {
		loggers[key] = levelMap[lg.GetLogLevel()]
	}
	return loggers
}

// GetOrCreateErrorLogger returns a ErrorLogger based on the output(p).
// If Logger not exists, and create function is not nil, creates a new logger
func (mng *ErrorLoggerManager) GetOrCreateErrorLogger(p string, level log.Level, f CreateErrorLoggerFunc) (log.ErrorLogger, error) {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	if lg, ok := mng.managers[p]; ok {
		return lg, nil
	}
	// only find exists
	if f == nil {
		return nil, ErrNoLoggerFound
	}

	// if logLevelControl log level is higher than level input
	// use logLevelControl to limit log level
	if mng.withLogLevelControl && mng.logLevelControl < level {
		level = mng.logLevelControl
	}

	lg, err := f(p, level)
	if err != nil {
		return nil, err
	}
	mng.managers[p] = lg

	if mng.disabled {
		lg.Toggle(true)
	}

	return lg, nil
}

func (mng *ErrorLoggerManager) SetLogLevelControl(level log.Level) {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	// save logLevelControl
	mng.withLogLevelControl = true
	mng.logLevelControl = level
}

func (mng *ErrorLoggerManager) DisableLogLevelControl() {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()

	mng.withLogLevelControl = false
	mng.logLevelControl = log.RAW
}

func (mng *ErrorLoggerManager) SetAllErrorLoggerLevel(level log.Level) {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	// check logLevelControl
	if mng.withLogLevelControl && mng.logLevelControl < level {
		level = mng.logLevelControl
	}
	for _, lg := range mng.managers {
		lg.SetLogLevel(level)
	}
}

func (mng *ErrorLoggerManager) Disable() {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()

	mng.disabled = true
	for _, lg := range mng.managers {
		lg.Toggle(true)
	}
}

func (mng *ErrorLoggerManager) Enable() {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()

	mng.disabled = false
	for _, lg := range mng.managers {
		lg.Toggle(false)
	}
}

// Default Export Functions
func GetErrorLoggerManagerInstance() *ErrorLoggerManager {
	return errorLoggerManagerInstance
}

// GetOrCreateDefaultErrorLogger used default create function
func GetOrCreateDefaultErrorLogger(p string, level log.Level) (log.ErrorLogger, error) {
	return errorLoggerManagerInstance.GetOrCreateErrorLogger(p, level, DefaultCreateErrorLoggerFunc)
}

// InitDefaultLogger inits a default logger
func InitDefaultLogger(output string, level log.Level) (err error) {
	DefaultLogger, err = GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return err
	}
	Proxy, err = CreateDefaultContextLogger(output, level)
	if err != nil {
		return err
	}
	// compatible for the mosn caller
	log.DefaultLogger = DefaultLogger
	log.DefaultContextLogger = Proxy
	return
}

//GetErrorLoggerInfo get the exists ErrorLogger
func GetErrorLoggersInfo() map[string]string {
	return errorLoggerManagerInstance.GetAllErrorLogger()
}

// UpdateErrorLoggerLevel updates the exists ErrorLogger's Level
func UpdateErrorLoggerLevel(p string, level log.Level) bool {
	// we use a nil create function means just get exists logger
	if lg, _ := errorLoggerManagerInstance.GetOrCreateErrorLogger(p, 0, nil); lg != nil {
		lg.SetLogLevel(level)
		return true
	}
	return false
}

// ToggleLogger enable/disable the exists logger, include ErrorLogger and Logger
func ToggleLogger(p string, disable bool) bool {
	// find ErrorLogger
	if lg, _ := errorLoggerManagerInstance.GetOrCreateErrorLogger(p, 0, nil); lg != nil {
		lg.Toggle(disable)
		return true
	}
	return log.ToggleLogger(p, disable)
}
