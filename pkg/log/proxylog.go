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
	"strconv"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
	"mosn.io/pkg/variable"
)

// proxyLogger is a default implementation of ContextLogger
// context will add trace info into formatter
// we use proxyLogger to record proxy events.
type proxyLogger struct {
	log.ErrorLogger
}

func CreateDefaultContextLogger(output string, level log.Level) (log.ContextLogger, error) {
	lg, err := GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return nil, err
	}
	return &proxyLogger{
		ErrorLogger: lg,
	}, nil

}
func (l *proxyLogger) fomatter(ctx context.Context, format string) string {
	return traceInfo(ctx) + " " + format
}

func (l *proxyLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.INFO {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Infof(s, args...)
	}
}

func (l *proxyLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.DEBUG {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Debugf(s, args...)
	}
}

func (l *proxyLogger) Tracef(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.TRACE {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Tracef(s, args...)
	}
}

func (l *proxyLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.WARN {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Warnf(s, args...)
	}
}

func (l *proxyLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.ERROR {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Errorf(s, args...)
	}
}

func (l *proxyLogger) Alertf(ctx context.Context, alert string, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.ERROR {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Alertf(alert, s, args...)
	}
}

func (l *proxyLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.GetLogLevel() >= log.FATAL {
		s := l.fomatter(ctx, format)
		l.ErrorLogger.Fatalf(s, args...)
	}
}
func traceInfo(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	cid := "-"
	tid := "-"
	uid := "-"

	connId, err := variable.Get(ctx, types.VariableConnectionID) // uint64
	if err == nil && connId != nil {
		uConnId, ok := connId.(uint64)
		if ok {
			cid = strconv.FormatUint(uConnId, 10)
		}

	}
	traceId, err := variable.Get(ctx, types.VariableTraceId) // string
	if err == nil && traceId != nil {
		tid = traceId.(string)
	}
	upstreamConnectionId, err := variable.Get(ctx, types.VariableUpstreamConnectionID) // uint64
	if err == nil && upstreamConnectionId != nil {
		uConnId, ok := upstreamConnectionId.(uint64)
		if ok {
			uid = strconv.FormatUint(uConnId, 10)
		}
	}

	return "[" + cid + "," + uid + "," + tid + "]"
}
