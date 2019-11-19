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

import "sofastack.io/sofa-mosn/common/utils"

// errorLogger is a default implementation of ErrorLogger
// we use ErrorLogger to write common log message.
type errorLogger struct {
	*Logger
	level Level
}

func CreateDefaultErrorLogger(output string, level Level) (ErrorLogger, error) {
	lg, err := GetOrCreateLogger(output, nil)
	if err != nil {
		return nil, err
	}
	return &errorLogger{
		Logger: lg,
		level:  level,
	}, nil
}

// default logger error level format:
// {time} [{level}] [{error code}] {content}
// default error code is normal
const defaultErrorCode = "normal"

func (l *errorLogger) formatter(lvPre string, format string) string {
	return utils.CacheTime() + " " + lvPre + " " + format
}

func (l *errorLogger) codeFormatter(lvPre, errCode, format string) string {
	return utils.CacheTime() + " " + lvPre + " [" + errCode + "] " + format
}

func (l *errorLogger) Infof(format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= INFO {
		s := l.formatter(InfoPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *errorLogger) Debugf(format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= DEBUG {
		s := l.formatter(DebugPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *errorLogger) Warnf(format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= WARN {
		s := l.formatter(WarnPre, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *errorLogger) Errorf(format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= ERROR {
		s := l.codeFormatter(ErrorPre, defaultErrorCode, format)
		l.Logger.Printf(s, args...)
	}
}

func (l *errorLogger) Alertf(errkey ErrorKey, format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= ERROR {
		s := l.codeFormatter(ErrorPre, string(errkey), format)
		l.Logger.Printf(s, args...)

	}
}

func (l *errorLogger) Tracef(format string, args ...interface{}) {
	if l.Logger.disable {
		return
	}
	if l.level >= TRACE {
		s := l.formatter(TracePre, format)
		l.Logger.Printf(s, args...)
	}
}

// Fatal logger cannot be disabled
func (l *errorLogger) Fatalf(format string, args ...interface{}) {
	s := l.formatter(FatalPre, format)
	l.Logger.Fatalf(s, args...)
}

func (l *errorLogger) Fatal(args ...interface{}) {
	args = append([]interface{}{
		l.formatter(FatalPre, ""),
	}, args...)
	l.Logger.Fatal(args...)
}

func (l *errorLogger) Fatalln(args ...interface{}) {
	args = append([]interface{}{
		l.formatter(FatalPre, ""),
	}, args...)
	l.Logger.Fatalln(args...)
}

func (l *errorLogger) SetLogLevel(level Level) {
	l.level = level
}

func (l *errorLogger) GetLogLevel() Level {
	return l.level
}
