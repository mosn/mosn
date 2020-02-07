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
	"mosn.io/pkg/utils"
)

var DefaultLogger ErrorLogger

func init() {
	logger, err := GetOrCreateLogger("", nil)
	if err != nil {
		panic("init default logger error: " + err.Error())
	}
	DefaultLogger = &SimpleErrorLog{
		Logger: logger,
		Level:  INFO,
	}
}

type SimpleErrorLog struct {
	*Logger
	Formatter func(lv string, alert string, format string) string
	Level     Level
}

func DefaultFormatter(lv string, alert string, format string) string {
	if alert == "" {
		return utils.CacheTime() + " " + lv + " " + format
	}
	return utils.CacheTime() + " " + lv + " [" + alert + "] " + format
}

func (l *SimpleErrorLog) Alertf(alert string, format string, args ...interface{}) {
	if l.disable {
		return
	}
	if l.Level >= ERROR {
		s := l.Formatter(ErrorPre, alert, format)
		l.Printf(s, args...)
	}
}
func (l *SimpleErrorLog) levelf(lv string, format string, args ...interface{}) {
	if l.disable {
		return
	}
	fs := ""
	if l.Formatter != nil {
		fs = l.Formatter(lv, "", format)
	} else {
		fs = DefaultFormatter(lv, "", format)
	}
	l.Printf(fs, args...)
}

func (l *SimpleErrorLog) Infof(format string, args ...interface{}) {
	if l.Level >= INFO {
		l.levelf(InfoPre, format, args...)
	}
}

func (l *SimpleErrorLog) Debugf(format string, args ...interface{}) {
	if l.Level >= DEBUG {
		l.levelf(DebugPre, format, args...)
	}
}

func (l *SimpleErrorLog) Warnf(format string, args ...interface{}) {
	if l.Level >= WARN {
		l.levelf(WarnPre, format, args...)
	}
}

func (l *SimpleErrorLog) Errorf(format string, args ...interface{}) {
	if l.Level >= ERROR {
		l.levelf(ErrorPre, format, args...)
	}
}

func (l *SimpleErrorLog) Tracef(format string, args ...interface{}) {
	if l.Level >= TRACE {
		l.levelf(TracePre, format, args...)
	}
}

func (l *SimpleErrorLog) Fatalf(format string, args ...interface{}) {
	var s string
	if l.Formatter != nil {
		s = l.Formatter(FatalPre, "", format)
	} else {
		s = DefaultFormatter(FatalPre, "", format)
	}
	l.Logger.Fatalf(s, args...)
}

func (l *SimpleErrorLog) SetLogLevel(level Level) {
	l.Level = level
}

func (l *SimpleErrorLog) GetLogLevel() Level {
	return l.Level
}
