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
	"errors"
	"strconv"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

// proxyLogger is a default implementation of ContextLogger
// context will add trace info into formatter
// we use poxyLogger to record proxy events.
type proxyLogger struct {
	*errorLogger
}

func CreateDefaultContextLogger(output string, level log.Level) (log.ContextLogger, error) {
	lg, err := GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return nil, err
	}
	if l, ok := lg.(*errorLogger); ok {
		return &proxyLogger{l}, nil
	} else {
		return nil, errors.New("proxy logger should equal mosn default error log")
	}

}
func (l *proxyLogger) fomatter(ctx context.Context, lv, alert, format string) string {
	return log.DefaultFormatter(lv, alert, traceInfo(ctx)+" "+format)
}

func (l *proxyLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.INFO {
		s := l.fomatter(ctx, log.InfoPre, "", format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.DEBUG {
		s := l.fomatter(ctx, log.DebugPre, "", format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.WARN {
		s := l.fomatter(ctx, log.WarnPre, "", format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.ERROR {
		s := l.fomatter(ctx, log.ErrorPre, defaultErrorCode, format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Alertf(ctx context.Context, alert string, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.ERROR {
		s := l.fomatter(ctx, log.ErrorPre, alert, format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	if l.Disable() {
		return
	}
	if l.Level >= log.FATAL {
		s := l.fomatter(ctx, log.FatalPre, "", format)
		l.Logger.Fatalf(s, args...)
	}
}

func (l *proxyLogger) GetLogLevel() log.Level {
	return l.Level
}

func (l *proxyLogger) SetLogLevel(level log.Level) {
	l.Level = level
}

func traceInfo(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	cid := "-"
	tid := "-"

	connId := mosnctx.Get(ctx, types.ContextKeyConnectionID) // uint64
	if connId != nil {
		cid = strconv.FormatUint(connId.(uint64), 10)
	}
	traceId := mosnctx.Get(ctx, types.ContextKeyTraceId) // string
	if traceId != nil {
		tid = traceId.(string)
	}

	return "[" + cid + "," + tid + "]"
}
