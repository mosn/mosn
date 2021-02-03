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

import "context"

var DefaultContextLogger ContextLogger

func init() {
	logger, err := GetOrCreateLogger("", nil)
	if err != nil {
		panic("init default logger error: " + err.Error())
	}
	DefaultContextLogger = &SimpleContextLog{
		SimpleErrorLog: &SimpleErrorLog{
			Logger: logger,
			Level:  INFO,
		},
	}
}

// SimpleComtextLog is a wrapper of SimpleErrorLog
type SimpleContextLog struct {
	*SimpleErrorLog
}

func (l *SimpleContextLog) Infof(ctx context.Context, format string, args ...interface{}) {
	l.SimpleErrorLog.Infof(format, args...)
}

func (l *SimpleContextLog) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.SimpleErrorLog.Debugf(format, args...)
}

func (l *SimpleContextLog) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.SimpleErrorLog.Infof(format, args...)
}

func (l *SimpleContextLog) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.SimpleErrorLog.Infof(format, args...)
}

func (l *SimpleContextLog) Alertf(ctx context.Context, alert string, format string, args ...interface{}) {
	l.SimpleErrorLog.Infof(format, args...)
}

func (l *SimpleContextLog) Fatalf(ctx context.Context, format string, args ...interface{}) {
	l.SimpleErrorLog.Infof(format, args...)
}
