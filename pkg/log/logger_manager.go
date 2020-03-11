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
	managers map[string]log.ErrorLogger
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
	lg, err := f(p, level)
	if err != nil {
		return nil, err
	}
	mng.managers[p] = lg
	return lg, nil
}

func (mng *ErrorLoggerManager) SetAllErrorLoggerLevel(level log.Level) {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	for _, lg := range mng.managers {
		lg.SetLogLevel(level)
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
