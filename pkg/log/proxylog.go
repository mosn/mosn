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

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
	"strconv"
)

// proxyLogger is a default implementation of ProxyLogger
// we use ProxyLogger to record proxy events.
type proxyLogger struct {
	ErrorLogger
}

func CreateDefaultProxyLogger(output string, level Level) (ProxyLogger, error) {
	lg, err := GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return nil, err
	}
	return &proxyLogger{
		ErrorLogger: lg,
	}, nil
}

// trace logger format:
// {time} [{level}] [{connId},{traceId}] {content}
func (l *proxyLogger) formatter(ctx context.Context, format string) string {
	return traceInfo(ctx) + " " + format
}

func (l *proxyLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.ErrorLogger.Infof(l.formatter(ctx, format), args...)
}

func (l *proxyLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.ErrorLogger.Debugf(l.formatter(ctx, format), args...)
}

func (l *proxyLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.ErrorLogger.Warnf(l.formatter(ctx, format), args...)
}

func (l *proxyLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.ErrorLogger.Printf(l.formatter(ctx, format), args...)
}

func (l *proxyLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	l.ErrorLogger.Fatalf(l.formatter(ctx, format), args...)
}

func (l *proxyLogger) SetLogLevel(level Level) {
	l.ErrorLogger.SetLogLevel(level)
}

func (l *proxyLogger) GetLogLevel() Level {
	return l.ErrorLogger.GetLogLevel()
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
