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
	"mosn.io/pkg/log"
)

// CreateErrorLoggerFunc creates a ErrorLogger implementation by output and level
type CreateErrorLoggerFunc func(output string, level log.Level) (log.ErrorLogger, error)

// The default function to CreateErrorLoggerFunc
var DefaultCreateErrorLoggerFunc CreateErrorLoggerFunc = CreateDefaultErrorLogger

// Wrapper of pkg/log
// Level is an alias of log.Level
type Level = log.Level

const (
	FATAL Level = log.FATAL
	ERROR       = log.ERROR
	WARN        = log.WARN
	INFO        = log.INFO
	DEBUG       = log.DEBUG
	TRACE       = log.TRACE
	RAW         = log.RAW
)

func GetOrCreateLogger(output string, roller *log.Roller) (*log.Logger, error) {
	return log.GetOrCreateLogger(output, roller)
}

// GetLogBuffer is an alias for log.GetLogBuffer
var GetLogBuffer = log.GetLogBuffer

// LogBuffer is an alias for log.LogBuffer
// nolint
type LogBuffer = log.LogBuffer
