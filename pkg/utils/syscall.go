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

package utils

import (
	"os"
	"syscall"
	"time"
)

var (
	// keep the standard for recover
	standardStdoutFd, _ = syscall.Dup(int(os.Stdout.Fd()))
	standardStderrFd, _ = syscall.Dup(int(os.Stderr.Fd()))
)

// SetHijackStdPipeline hijacks stdout and stderr outputs into the file path
func SetHijackStdPipeline(filepath string, stdout, stderr bool) {
	files := []*os.File{}
	if stdout {
		files = append(files, os.Stdout)
	}
	if stderr {
		files = append(files, os.Stderr)
	}
	GoWithRecover(func() {
		ResetHjiackStdPipeline()
		setHijackFile(files, filepath)
	}, nil)
}

func ResetHjiackStdPipeline() {
	Dup(standardStdoutFd, int(os.Stdout.Fd()))
	Dup(standardStderrFd, int(os.Stderr.Fd()))
}

// setHijackFile hijacks the stdFile outputs into the new file
// the new file will be rotated each {hijackRotateInterval}, and we keep one old file
func setHijackFile(stdFiles []*os.File, newFilePath string) {
	hijack := func() {
		fp, err := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		for _, stdFile := range stdFiles {
			Dup(int(fp.Fd()), int(stdFile.Fd()))
		}
	}
	rotate := func(today string) {
		if err := os.Rename(newFilePath, newFilePath+"."+today); err != nil {
			return
		}
		hijack()
	}
	if len(stdFiles) > 0 {
		// call
		hijack()
		// rotate by day
		for {
			todayStr := time.Now().Format("2006-01-02")
			time.Sleep(nextDayDuration())
			rotate(todayStr)
		}
	}

}

// nextDayDuration returns the duration to next day
func nextDayDuration() time.Duration {
	now := time.Now()
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local) // use system location
	nextday := today.Add(24 * time.Hour)
	return nextday.Sub(now)
}
