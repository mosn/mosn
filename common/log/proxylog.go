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

	mosnctx "sofastack.io/sofa-mosn/common/context"
	"sofastack.io/sofa-mosn/common/utils"
)

// proxyLogger is a default implementation of ProxyLogger
// we use ProxyLogger to record proxy events.
type proxyLogger struct {
	*errorLogger
}

func CreateDefaultProxyLogger(output string, level Level) (ProxyLogger, error) {
	lg, err := GetOrCreateDefaultErrorLogger(output, level)
	if err != nil {
		return nil, err
	}
	return &proxyLogger{lg.(*errorLogger)}, nil
}

// trace logger format:
// {time} [{level}] [{connId},{traceId}] {content}
func (l *proxyLogger) formatter(ctx context.Context, lvPre string, format string) string {
	return utils.CacheTime() + " " + lvPre + " " + traceInfo(ctx) + " " + format
}

func (l *proxyLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= INFO {
		s := l.formatter(ctx, InfoPre, format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= DEBUG {
		s := l.formatter(ctx, DebugPre, format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= WARN {
		s := l.formatter(ctx, WarnPre, format)
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= ERROR {
		s := utils.CacheTime() + " " + ErrorPre + " [" + defaultErrorCode + "] " + traceInfo(ctx) + " " + format
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Alertf(ctx context.Context, errkey ErrorKey, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= ERROR {
		s := utils.CacheTime() + " " + ErrorPre + " [" + string(errkey) + "] " + traceInfo(ctx) + " " + format
		l.Printf(s, args...)
	}
}

func (l *proxyLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.level >= FATAL {
		s := l.formatter(ctx, FatalPre, format)
		l.Logger.Fatalf(s, args...)
	}
}

func traceInfo(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	cid := "-"
	tid := "-"

	connId := mosnctx.Get(ctx, mosnctx.ContextKeyConnectionID) // uint64
	if connId != nil {
		cid = strconv.FormatUint(connId.(uint64), 10)
	}
	traceId := mosnctx.Get(ctx, mosnctx.ContextKeyTraceId) // string
	if traceId != nil {
		tid = traceId.(string)
	}

	return "[" + cid + "," + tid + "]"
}
