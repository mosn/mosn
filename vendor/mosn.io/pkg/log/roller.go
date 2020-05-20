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
	"io"
	"path/filepath"
	"strconv"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	// defaultRoller is roller by one day
	defaultRoller = Roller{MaxTime: defaultRotateTime}

	// lumberjacks maps log filenames to the logger
	// that is being used to keep them rolled/maintained.
	lumberjacks = make(map[string]*lumberjack.Logger)

	errInvalidRollerParameter = errors.New("invalid roller parameter")
)

const (
	// max royate time is 24 hours
	maxRotateHour = 24
	// defaultRotateTime is 24 hours
	defaultRotateTime = 24 * 60 * 60
	// defaultRotateSize is 100 MB.
	defaultRotateSize = 1000
	// defaultRotateAge is 7 days.
	defaultRotateAge = 7
	// defaultRotateKeep is 10 files.
	defaultRotateKeep = 10

	directiveRotateTime     = "time"
	directiveRotateSize     = "size"
	directiveRotateAge      = "age"
	directiveRotateKeep     = "keep"
	directiveRotateCompress = "compress"
)

// roller implements a type that provides a rolling logger.
type Roller struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
	LocalTime  bool
	// roller rotate time, if the MAxTime is configured, ignore the others config
	MaxTime int64
}

// GetLogWriter returns an io.Writer that writes to a rolling logger.
// This should be called only from the main goroutine (like during
// server setup) because this method is not thread-safe; it is careful
// to create only one log writer per log file, even if the log file
// is shared by different sites or middlewares. This ensures that
// rolling is synchronized, since a process (or multiple processes)
// should not create more than one roller on the same file at the
// same time. See issue #1363.
func (l Roller) GetLogWriter() io.Writer {
	absPath, err := filepath.Abs(l.Filename)
	if err != nil {
		absPath = l.Filename // oh well, hopefully they're consistent in how they specify the filename
	}
	lj, has := lumberjacks[absPath]
	if !has {
		lj = &lumberjack.Logger{
			Filename:   l.Filename,
			MaxSize:    l.MaxSize,
			MaxAge:     l.MaxAge,
			MaxBackups: l.MaxBackups,
			Compress:   l.Compress,
			LocalTime:  l.LocalTime,
		}
		lumberjacks[absPath] = lj
	}

	return lj
}

// InitDefaultRoller
func InitGlobalRoller(roller string) error {
	r, err := ParseRoller(roller)
	if err != nil {
		return err
	}
	defaultRoller = *r
	return nil
}

// DefaultRoller will roll logs by default.
func DefaultRoller() *Roller {
	return &Roller{
		MaxSize:    defaultRotateSize,
		MaxAge:     defaultRotateAge,
		MaxBackups: defaultRotateKeep,
		Compress:   false,
		LocalTime:  true,
	}
}

// ParseRoller parses roller contents out of c.
func ParseRoller(what string) (*Roller, error) {
	var err error
	var value int
	roller := DefaultRoller()
	for _, args := range strings.Split(what, " ") {
		v := strings.Split(args, "=")
		if len(v) != 2 {
			err = errInvalidRollerParameter
			break
		}
		switch v[0] {
		case directiveRotateTime:
			value, err = strconv.Atoi(v[1])
			if err != nil {
				break
			}
			if value > maxRotateHour {
				value = maxRotateHour
			}
			roller.MaxTime = int64(value) * 60 * 60
		case directiveRotateSize:
			value, err = strconv.Atoi(v[1])
			if err != nil {
				break
			}
			roller.MaxSize = value
		case directiveRotateAge:
			value, err = strconv.Atoi(v[1])
			if err != nil {
				break
			}
			roller.MaxAge = value
		case directiveRotateKeep:
			value, err = strconv.Atoi(v[1])
			if err != nil {
				break
			}
			roller.MaxBackups = value
		case directiveRotateCompress:
			if v[1] == "on" {
				roller.Compress = true
			} else if v[1] == "off" {
				roller.Compress = false
			} else {
				err = errInvalidRollerParameter
			}
		default:
			err = errInvalidRollerParameter
		}
	}
	if err != nil {
		return nil, err
	}

	return roller, nil
}

// IsLogRollerSubdirective is true if the subdirective is for the log roller.
func IsLogRollerSubdirective(subdir string) bool {
	return subdir == directiveRotateSize ||
		subdir == directiveRotateAge ||
		subdir == directiveRotateKeep ||
		subdir == directiveRotateCompress
}
