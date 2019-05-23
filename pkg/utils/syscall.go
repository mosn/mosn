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
	hijackRotateInterval = 24 * time.Hour
	// keep the standard for recover
	standardStdoutFd, _ = syscall.Dup(int(os.Stdout.Fd()))
	standardStderrFd, _ = syscall.Dup(int(os.Stderr.Fd()))
)

// SetHijackStdPipeline hijacks stdout and stderr outputs into the file path
func SetHijackStdPipeline(stdout string, stderr string) {
	if stdout != "" {
		GoWithRecover(func() {
			setHijackFile(os.Stdout, stdout)
		}, nil)
	}
	if stderr != "" {
		GoWithRecover(func() {
			setHijackFile(os.Stderr, stderr)
		}, nil)
	}
}

func ResetHjiackStdPipeline() {
	syscall.Dup2(standardStdoutFd, int(os.Stdout.Fd()))
	syscall.Dup2(standardStderrFd, int(os.Stderr.Fd()))

}

// setHijackFile hijacks the stdFile outputs into the new file
// the new file will be rotated each {hijackRotateInterval}, and we keep one old file
func setHijackFile(stdFile *os.File, newFilePath string) {
	hijack := func() {
		fp, err := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		syscall.Dup2(int(fp.Fd()), int(stdFile.Fd()))
	}
	// call
	hijack()
	// rotate
	ticker := time.NewTicker(hijackRotateInterval)
	for c := range ticker.C {
		_ = c
		if err := os.Rename(newFilePath, newFilePath+".old"); err != nil {
			continue
		}
		hijack()
	}

}
