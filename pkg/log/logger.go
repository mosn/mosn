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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/hashicorp/go-syslog"
)

// Log Instance
var (
	DefaultLogger        *logger
	StartLogger          *logger
	remoteSyslogPrefixes = map[string]string{
		"syslog+tcp://": "tcp",
		"syslog+udp://": "udp",
		"syslog://":     "udp",
	}

	//due to the fact that multiple access log owned by different listener can point to same log path
	//make logger instance for each log path unique, and can be shared by different access log
	// AccessLog x(path:/home/a, format:foo) -> RealLog a
	// AccessLog y(path:/home/a, format:bar) -> RealLog a
	// AccessLog z(path:/home/b)             -> RealLog b
	loggers []*logger

	lastTime atomic.Value

	defaultRoller *Roller

	//defaultRollerTime is one day
	defaultRollerTime int64 = 24 * 60 * 60
)

// time cache
type timeCache struct {
	t int64
	s string
}

func init() {
	//use console  as start logger
	StartLogger, _ = NewLogger("", INFO)
	// default as start before Init
	DefaultLogger = StartLogger
}

// Logger
type logger struct {
	Output string
	Level  Level
	roller *Roller
	writer io.Writer
	create time.Time

	reopenChan      chan struct{}
	closeChan       chan struct{}
	writeBufferChan chan types.IoBuffer
}

// InitDefaultLogger
// start default logger
func InitDefaultLogger(output string, level Level) error {
	var err error
	DefaultLogger, err = NewLogger(output, level)

	return err
}

// InitDefaultRoller
func InitDefaultRoller(roller string) error {
	var err error
	defaultRoller, err = ParseRoller(roller)
	return err
}

// ByContext
// Get default logger by context
func ByContext(ctx context.Context) Logger {
	if ctx != nil {
		if logger := ctx.Value(types.ContextKeyLogger); logger != nil {
			return logger.(Logger)
		}
	}

	if DefaultLogger == nil {
		InitDefaultLogger("", DEBUG)
	}

	return DefaultLogger
}

// GetLoggerInstance
// get logger instance which has the same 'output' and 'level'
func GetLoggerInstance(output string, level Level) (Logger, error) {
	for _, logger := range loggers {
		if logger.Output == output && logger.Level == level {
			return logger, nil
		}
	}

	return NewLogger(output, level)
}

// NewLogger
func NewLogger(output string, level Level) (*logger, error) {
	logger := &logger{
		Output:          output,
		Level:           level,
		roller:          defaultRoller,
		writeBufferChan: make(chan types.IoBuffer, 1000),
		reopenChan:      make(chan struct{}),
		closeChan:       make(chan struct{}),
	}

	loggers = append(loggers, logger)
	return logger, logger.Start()
}

func (l *logger) Start() error {
	var err error

selectwriter:
	switch l.Output {
	case "", "stderr", "/dev/stderr":
		l.writer = os.Stderr
	case "stdout", "/dev/stdout":
		l.writer = os.Stdout
	case "syslog":
		l.writer, err = gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "mosn")
		if err != nil {
			return err
		}
	default:
		if address := parseSyslogAddress(l.Output); address != nil {
			l.writer, err = gsyslog.DialLogger(address.network, address.address, gsyslog.LOG_ERR, "LOCAL0", "mosn")

			if err != nil {
				return err
			}

			break selectwriter
		}

		var file *os.File
		var stat os.FileInfo

		//create parent dir if not exists
		err := os.MkdirAll(filepath.Dir(l.Output), 0755)

		fmt.Println(err)

		file, err = os.OpenFile(l.Output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		if l.roller != nil {
			file.Close()
			l.roller.Filename = l.Output
			l.writer = l.roller.GetLogWriter()
		} else {
			stat, err = file.Stat()
			if err != nil {
				return err
			}
			l.create = stat.ModTime()
			l.writer = file
		}
	}

	go l.handler()

	return nil
}

func (l *logger) handler() {
	defer func() {
		if p := recover(); p != nil {
			debug.PrintStack()
			go l.handler()
		}
	}()

	var buf types.IoBuffer
	for {
		select {
		case <-l.reopenChan:
			l.reopen()
			return
		case <-l.closeChan:
			for {
				select {
				case buf = <-l.writeBufferChan:
					buf.WriteTo(l)
					buffer.PutIoBuffer(buf)
				default:
					l.close()
					return
				}
			}
		case buf = <-l.writeBufferChan:
			for i := 0; i < 20; i++ {
				select {
				case b := <-l.writeBufferChan:
					buf.Write(b.Bytes())
					buffer.PutIoBuffer(b)
				default:
					break
				}
			}
		}
		buf.WriteTo(l)
		buffer.PutIoBuffer(buf)
	}
}

func (l *logger) Print(buf types.IoBuffer, discard bool) error {
	select {
	case l.writeBufferChan <- buf:
	default:
		// todo: configurable
		if discard {
			return types.ErrChanFull
		} else {
			l.writeBufferChan <- buf
		}
	}
	return nil
}

func (l *logger) Println(args ...interface{}) {
	s := fmt.Sprintln(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

func (l *logger) Printf(format string, args ...interface{}) {
	s := fmt.Sprintf(logTime()+" "+format, args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

func (l *logger) Infof(format string, args ...interface{}) {
	if l.Level >= INFO {
		l.Printf(InfoPre+format, args...)
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.Level >= DEBUG {
		l.Printf(DebugPre+format, args...)
	}
}

func (l *logger) Warnf(format string, args ...interface{}) {
	if l.Level >= WARN {
		l.Printf(WarnPre+format, args...)
	}
}

func (l *logger) Errorf(format string, args ...interface{}) {
	if l.Level >= ERROR {
		l.Printf(ErrorPre+format, args...)
	}
}

func (l *logger) Tracef(format string, args ...interface{}) {
	if l.Level >= TRACE {
		l.Printf(TracePre+format, args...)
	}
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	s := fmt.Sprintf(logTime()+" "+FatalPre+format, args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *logger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(logTime() + " " + FatalPre)
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *logger) Fatalln(args ...interface{}) {
	s := fmt.Sprintln(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(logTime() + " " + FatalPre)
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *logger) Write(p []byte) (n int, err error) {
	// // default roller by daily
	if l.roller == nil {
		if !l.create.IsZero() {
			now := time.Now()
			if l.create.Unix()/defaultRollerTime != now.Unix()/defaultRollerTime {
				if err = os.Rename(l.Output, l.Output+"."+l.create.Format("2006-01-02")); err != nil {
					return 0, err
				}
				l.create = now
				go l.Reopen()
			}
		}
	}
	return l.writer.Write(p)
}

func (l *logger) Close() error {
	l.closeChan <- struct{}{}
	return nil
}

func (l *logger) close() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		err := closer.Close()
		return err
	}

	return nil
}

func (l *logger) Reopen() error {
	l.reopenChan <- struct{}{}
	return nil
}

func (l *logger) reopen() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		err := closer.Close()

		if err := l.Start(); err != nil {
			return err
		}
		return err
	}

	return nil
}

type syslogAddress struct {
	network string
	address string
}

func parseSyslogAddress(location string) *syslogAddress {
	for prefix, network := range remoteSyslogPrefixes {
		if strings.HasPrefix(location, prefix) {
			return &syslogAddress{
				network: network,
				address: strings.TrimPrefix(location, prefix),
			}
		}
	}

	return nil
}

// Reopen all logger
func Reopen() error {
	for _, logger := range loggers {
		if err := logger.Reopen(); err != nil {
			return err
		}
	}

	return nil
}

// CloseAll logger
func CloseAll() error {
	for _, logger := range loggers {
		if err := logger.Close(); err != nil {
			return err
		}
	}

	return nil
}

func logTime() string {
	var s string
	t := time.Now()
	now := t.Unix()
	value := lastTime.Load()
	if value != nil {
		last := value.(*timeCache)
		if now <= last.t {
			s = last.s
		}
	}
	if s == "" {
		s = t.Format("2006/01/02 15:04:05")
		lastTime.Store(&timeCache{now, s})
	}
	return s
}
