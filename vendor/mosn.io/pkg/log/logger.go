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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	gsyslog "github.com/hashicorp/go-syslog"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

var (
	// localOffset is offset in seconds east of UTC
	_, localOffset = time.Now().Zone()
	// error
	ErrReopenUnsupported = errors.New("reopen unsupported")

	remoteSyslogPrefixes = map[string]string{
		"syslog+tcp://": "tcp",
		"syslog+udp://": "udp",
		"syslog://":     "udp",
	}
)

// Logger is a basic sync logger implement, contains unexported fields
// The Logger Function contains:
// Print(buffer buffer.IoBuffer, discard bool) error
// Printf(format string, args ...interface{})
// Println(args ...interface{})
// Fatalf(format string, args ...interface{})
// Fatal(args ...interface{})
// Fatalln(args ...interface{})
// Close() error
// Reopen() error
// Toggle(disable bool)
type Logger struct {
	// output is the log's output path
	// if output is empty(""), it is equals to stderr
	output string
	// writer writes the log, created by output
	writer io.Writer
	// roller rotates the log, if the output is a file path
	roller *Roller
	// disable presents the logger state. if disable is true, the logger will write nothing
	// the default value is false
	disable bool
	// implementation elements
	create          time.Time
	once            sync.Once
	stopRotate      chan struct{}
	reopenChan      chan struct{}
	closeChan       chan struct{}
	writeBufferChan chan buffer.IoBuffer
}

// loggers keeps all Logger we created
// key is output, same output reference the same Logger
var loggers sync.Map // map[string]*Logger

func ToggleLogger(p string, disable bool) bool {
	// find Logger
	if lg, ok := loggers.Load(p); ok {
		lg.(*Logger).Toggle(disable)
		return true
	}
	return false
}

// Reopen all logger
func Reopen() (err error) {
	loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		err = logger.Reopen()
		if err != nil {
			return false
		}
		return true
	})
	return
}

// CloseAll logger
func CloseAll() (err error) {
	loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		err = logger.Close()
		if err != nil {
			return false
		}
		return true
	})
	return
}

// ClearAll created logger, just for test
func ClearAll() {
	loggers = sync.Map{}
}

func GetOrCreateLogger(output string, roller *Roller) (*Logger, error) {
	if lg, ok := loggers.Load(output); ok {
		return lg.(*Logger), nil
	}

	if roller == nil {
		roller = &defaultRoller
	}

	lg := &Logger{
		output:          output,
		roller:          roller,
		writeBufferChan: make(chan buffer.IoBuffer, 500),
		reopenChan:      make(chan struct{}),
		closeChan:       make(chan struct{}),
		stopRotate:      make(chan struct{}),
		// writer and create will be setted in start()
	}
	err := lg.start()
	if err == nil { // only keeps start success logger
		loggers.Store(output, lg)
	}
	return lg, err
}

func (l *Logger) start() error {
	switch l.output {
	case "", "stderr", "/dev/stderr":
		l.writer = os.Stderr
	case "stdout", "/dev/stdout":
		l.writer = os.Stdout
	case "syslog":
		writer, err := gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "mosn")
		if err != nil {
			return err
		}
		l.writer = writer
	default:
		if address := parseSyslogAddress(l.output); address != nil {
			writer, err := gsyslog.DialLogger(address.network, address.address, gsyslog.LOG_ERR, "LOCAL0", "mosn")
			if err != nil {
				return err
			}
			l.writer = writer
		} else { // write to file
			if err := os.MkdirAll(filepath.Dir(l.output), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(l.output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				return err
			}
			if l.roller.MaxTime == 0 {
				file.Close()
				l.roller.Filename = l.output
				l.writer = l.roller.GetLogWriter()
			} else {
				// time.Now() faster than reported timestamps from filesystem (https://github.com/golang/go/issues/33510)
				// init logger
				if l.create.IsZero() {
					stat, err := file.Stat()
					if err != nil {
						return err
					}
					l.create = stat.ModTime()
				} else {
					l.create = time.Now()
				}
				l.writer = file
				l.once.Do(l.startRotate) // start rotate, only once
			}
		}
	}
	// TODO: recover?
	go l.handler()
	return nil
}

func (l *Logger) handler() {
	defer func() {
		if p := recover(); p != nil {
			debug.PrintStack()
			// TODO: recover?
			go l.handler()
		}
	}()
	var buf buffer.IoBuffer
	for {
		select {
		case <-l.reopenChan:
			// reopen is used for roller
			err := l.reopen()
			if err == nil {
				return
			}
			fmt.Printf("%s reopen failed : %v\n", l.output, err)
		case <-l.closeChan:
			// flush all buffers before close
			// make sure all logs are outputed
			// a closed logger can not write anymore
			for {
				select {
				case buf = <-l.writeBufferChan:
					buf.WriteTo(l)
					buffer.PutIoBuffer(buf)
				default:
					l.stop()
					close(l.stopRotate)
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
			buf.WriteTo(l)
			buffer.PutIoBuffer(buf)
		}
		runtime.Gosched()
	}
}

func (l *Logger) stop() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		err := closer.Close()
		return err
	}

	return nil
}

func (l *Logger) reopen() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return ErrReopenUnsupported
	}
	if closer, ok := l.writer.(io.WriteCloser); ok {
		err := closer.Close()
		if err != nil {
			return err
		}
		return l.start()
	}
	return ErrReopenUnsupported
}

var ErrChanFull = errors.New("channel is full")

// Print writes the final buffere to the buffer chan
// if discard is true and the buffer is full, returns an error
func (l *Logger) Print(buf buffer.IoBuffer, discard bool) error {
	if l.disable {
		// free the buf
		buffer.PutIoBuffer(buf)
		return nil
	}
	select {
	case l.writeBufferChan <- buf:
	default:
		// todo: configurable
		if discard {
			return ErrChanFull
		} else {
			l.writeBufferChan <- buf
		}
	}
	return nil
}

func (l *Logger) Println(args ...interface{}) {
	if l.disable {
		return
	}
	s := fmt.Sprintln(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.disable {
		return
	}
	s := fmt.Sprintf(format, args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

// Fatal cannot be disabled
func (l *Logger) Fatalf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	buf.WriteString("\n")
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) Fatalln(args ...interface{}) {
	s := fmt.Sprintln(args...)
	buf := buffer.GetIoBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) startRotate() {
	utils.GoWithRecover(func() {
		// roller not by time
		if l.create.IsZero() {
			return
		}
		var interval time.Duration
		// check need to rotate right now
		now := time.Now()
		if now.Sub(l.create) > time.Duration(l.roller.MaxTime)*time.Second {
			interval = 0
		} else {
			// caculate the next time need to rotate
			interval = time.Duration(l.roller.MaxTime-(now.Unix()+int64(localOffset))%l.roller.MaxTime) * time.Second
		}
		doRotate(l, interval)
	}, func(r interface{}) {
		l.startRotate()
	})
}

var doRotate func(l *Logger, interval time.Duration) = doRotateFunc

func doRotateFunc(l *Logger, interval time.Duration) {
	for {
		select {
		case <-l.stopRotate:
			return
		case <-time.After(interval):
			now := time.Now()
			// ignore the rename error, in case the l.output is deleted
			if l.roller.MaxTime == defaultRotateTime {
				os.Rename(l.output, l.output+"."+l.create.Format("2006-01-02"))
			} else {
				os.Rename(l.output, l.output+"."+l.create.Format("2006-01-02_15"))
			}
			l.create = now
			go l.Reopen()

			if interval == 0 { // recaculate interval
				interval = time.Duration(l.roller.MaxTime-(now.Unix()+int64(localOffset))%l.roller.MaxTime) * time.Second
			} else {
				interval = time.Duration(l.roller.MaxTime) * time.Second
			}
		}
	}
}

func (l *Logger) Write(p []byte) (n int, err error) {
	return l.writer.Write(p)
}

func (l *Logger) Close() error {
	l.closeChan <- struct{}{}
	return nil
}

func (l *Logger) Reopen() error {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()
	l.reopenChan <- struct{}{}
	return nil
}

func (l *Logger) Toggle(disable bool) {
	l.disable = disable
}

func (l *Logger) Disable() bool {
	return l.disable
}

// syslogAddress
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
