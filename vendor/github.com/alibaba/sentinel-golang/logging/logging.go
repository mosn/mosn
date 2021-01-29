// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents the level of logging.
type Level uint8

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

const (
	// RecordLogFileName represents the default file name of the record log.
	RecordLogFileName = "sentinel-record.log"
	GlobalCallerDepth = 4

	defaultLogMsgBufferSize = 256
)

var (
	DefaultDirName = filepath.Join("logs", "csp")
)

var (
	globalLogLevel = InfoLevel
	globalLogger   = NewConsoleLogger()

	FrequentErrorOnce = &sync.Once{}
)

// GetGlobalLoggerLevel gets the Sentinel log level
func GetGlobalLoggerLevel() Level {
	return globalLogLevel
}

// ResetGlobalLoggerLevel sets the Sentinel log level
// Note: this function is not thread-safe.
func ResetGlobalLoggerLevel(l Level) {
	globalLogLevel = l
}

// GetGlobalLogger gets the Sentinel global logger
func GetGlobalLogger() Logger {
	return globalLogger
}

// ResetGlobalLogger sets the Sentinel global logger
// Note: this function is not thread-safe.
func ResetGlobalLogger(log Logger) error {
	if log == nil {
		return errors.New("nil logger")
	}
	globalLogger = log
	return nil
}

func NewConsoleLogger() Logger {
	return &DefaultLogger{
		log: log.New(os.Stdout, "", 0),
	}
}

// outputFile is the full path(absolute path)
func NewSimpleFileLogger(filepath string) (Logger, error) {
	logFile, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	return &DefaultLogger{
		log: log.New(logFile, "", 0),
	}, nil
}

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	DebugEnabled() bool

	// Info logs a non-error message with the given key/value pairs as context.
	//
	// The msg argument should be used to add some constant description to
	// the log line.  The key/value pairs can then be used to add additional
	// variable information.  The key/value pairs should alternate string
	// keys and arbitrary values.
	Info(msg string, keysAndValues ...interface{})
	InfoEnabled() bool

	Warn(msg string, keysAndValues ...interface{})
	WarnEnabled() bool

	Error(err error, msg string, keysAndValues ...interface{})
	ErrorEnabled() bool
}

// sentinel general logger
type DefaultLogger struct {
	log *log.Logger
}

func caller(depth int) (file string, line int) {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = "???"
		line = 0
	}

	// extract
	if osType := runtime.GOOS; osType == "windows" {
		file = strings.ReplaceAll(file, "\\", "/")
	}
	idx := strings.LastIndex(file, "/")
	file = file[idx+1:]
	return
}

// toSafeJSONString converts to valid JSON string, as the original string may contain '\\', '\n', '\r', '\t' and so on.
func toSafeJSONString(s string) []byte {
	if data, err := json.Marshal(json.RawMessage(s)); err == nil {
		return data
	} else {
		return []byte("\"" + s + "\"")
	}
}

func AssembleMsg(depth int, logLevel, msg string, err error, keysAndValues ...interface{}) string {
	sb := strings.Builder{}
	sb.Grow(defaultLogMsgBufferSize)

	file, line := caller(depth)
	timeStr := time.Now().Format("2006-01-02 15:04:05.520")
	caller := fmt.Sprintf("%s:%d", file, line)
	sb.WriteString("{")

	sb.WriteByte('"')
	sb.WriteString("timestamp")
	sb.WriteByte('"')
	sb.WriteByte(':')
	sb.WriteByte('"')
	sb.WriteString(timeStr)
	sb.WriteByte('"')
	sb.WriteByte(',')

	sb.WriteByte('"')
	sb.WriteString("caller")
	sb.WriteByte('"')
	sb.WriteByte(':')
	sb.WriteByte('"')
	sb.WriteString(caller)
	sb.WriteByte('"')
	sb.WriteByte(',')

	sb.WriteByte('"')
	sb.WriteString("logLevel")
	sb.WriteByte('"')
	sb.WriteByte(':')
	sb.WriteByte('"')
	sb.WriteString(logLevel)
	sb.WriteByte('"')
	sb.WriteByte(',')

	sb.WriteByte('"')
	sb.WriteString("msg")
	sb.WriteByte('"')
	sb.WriteByte(':')
	data := toSafeJSONString(msg)
	sb.Write(data)

	kvLen := len(keysAndValues)
	if kvLen&1 != 0 {
		sb.WriteByte(',')
		sb.WriteByte('"')
		sb.WriteString("kvs")
		sb.WriteByte('"')
		sb.WriteByte(':')
		s := fmt.Sprintf("%+v", keysAndValues)
		data := toSafeJSONString(s)
		sb.Write(data)
	} else if kvLen != 0 {
		for i := 0; i < kvLen; {
			k := keysAndValues[i]
			v := keysAndValues[i+1]
			kStr, kIsStr := k.(string)
			if !kIsStr {
				kStr = fmt.Sprintf("%+v", k)
			}
			sb.WriteByte(',')
			data := toSafeJSONString(kStr)
			sb.Write(data)
			sb.WriteByte(':')
			switch v.(type) {
			case string:
				data := toSafeJSONString(v.(string))
				sb.Write(data)
			case error:
				data := toSafeJSONString(v.(error).Error())
				sb.Write(data)
			default:
				if vbs, err := json.Marshal(v); err != nil {
					s := fmt.Sprintf("%+v", v)
					data := toSafeJSONString(s)
					sb.Write(data)
				} else {
					sb.Write(vbs)
				}
			}
			i = i + 2
		}
	}
	sb.WriteByte('}')
	if err != nil {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%+v", err))
	}
	return sb.String()
}

func (l *DefaultLogger) Debug(msg string, keysAndValues ...interface{}) {
	if !l.DebugEnabled() {
		return
	}
	l.log.Print(AssembleMsg(GlobalCallerDepth, "DEBUG", msg, nil, keysAndValues...))
}

func (l *DefaultLogger) DebugEnabled() bool {
	return DebugLevel >= globalLogLevel
}

func (l *DefaultLogger) Info(msg string, keysAndValues ...interface{}) {
	if !l.InfoEnabled() {
		return
	}
	l.log.Print(AssembleMsg(GlobalCallerDepth, "INFO", msg, nil, keysAndValues...))
}

func (l *DefaultLogger) InfoEnabled() bool {
	return InfoLevel >= globalLogLevel
}

func (l *DefaultLogger) Warn(msg string, keysAndValues ...interface{}) {
	if !l.WarnEnabled() {
		return
	}

	l.log.Print(AssembleMsg(GlobalCallerDepth, "WARNING", msg, nil, keysAndValues...))
}

func (l *DefaultLogger) WarnEnabled() bool {
	return WarnLevel >= globalLogLevel
}

func (l *DefaultLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if !l.ErrorEnabled() {
		return
	}
	l.log.Print(AssembleMsg(GlobalCallerDepth, "ERROR", msg, err, keysAndValues...))
}

func (l *DefaultLogger) ErrorEnabled() bool {
	return ErrorLevel >= globalLogLevel
}

func Debug(msg string, keysAndValues ...interface{}) {
	globalLogger.Debug(msg, keysAndValues...)
}

func DebugEnabled() bool {
	return globalLogger.DebugEnabled()
}

func Info(msg string, keysAndValues ...interface{}) {
	globalLogger.Info(msg, keysAndValues...)
}

func InfoEnabled() bool {
	return globalLogger.InfoEnabled()
}

func Warn(msg string, keysAndValues ...interface{}) {
	globalLogger.Warn(msg, keysAndValues...)
}

func WarnEnabled() bool {
	return globalLogger.WarnEnabled()
}

func Error(err error, msg string, keysAndValues ...interface{}) {
	globalLogger.Error(err, msg, keysAndValues...)
}

func ErrorEnabled() bool {
	return globalLogger.ErrorEnabled()
}
