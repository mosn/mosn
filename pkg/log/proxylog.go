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

	"github.com/alipay/sofa-mosn/pkg/types"

	"strconv"

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
)

// proxyLogger is a default implementation of ProxyLogger
// we use ProxyLogger to record proxy events.
type proxyLogger struct {
	Logger  *errorLogger
	disable bool
}

func CreateDefaultProxyLogger(output string, level Level) (ProxyLogger, error) {
	lg, err := GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return nil, err
	}
	return &proxyLogger{
		Logger: lg.(*errorLogger),
	}, nil
}

// trace logger format:
// {time} [{level}] [{connId},{traceId}] {content}
func (l *proxyLogger) formatter(ctx context.Context, lvPre string, format string) string {
	return logTime() + " " + lvPre + " " + traceInfo(ctx) + " " + format
}

func (l *proxyLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.GetLogLevel() >= INFO {
		s := l.formatter(ctx, InfoPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *proxyLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.GetLogLevel() >= DEBUG {
		s := l.formatter(ctx, DebugPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *proxyLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.GetLogLevel() >= WARN {
		s := l.formatter(ctx, WarnPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *proxyLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.GetLogLevel() >= ERROR {
		s := l.formatter(ctx, ErrorPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *proxyLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.GetLogLevel() >= FATAL {
		s := l.formatter(ctx, FatalPre, format)
		l.Logger.Fatalf(s, args...)
	}
}

func (l *proxyLogger) SetLogLevel(level Level) {
	l.Logger.SetLogLevel(level)
}

func (l *proxyLogger) GetLogLevel() Level {
	return l.Logger.GetLogLevel()
}

// Toggle disable/enable the logger
func (l *proxyLogger) Toggle(disable bool) {
	l.Logger.Toggle(disable)
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
