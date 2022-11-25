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

package pid

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

var (
	pidFile string
)

func SetPid(pid string) {
	if pid == "" {
		pidFile = types.MosnPidDefaultFileName
	} else {
		if err := os.MkdirAll(filepath.Dir(pid), 0755); err != nil {
			pidFile = types.MosnPidDefaultFileName
		} else {
			pidFile = pid
		}
	}
	WritePidFile()
}

func WritePidFile() (err error) {
	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")

	if err = ioutil.WriteFile(pidFile, pid, 0644); err != nil {
		log.DefaultLogger.Errorf("write pid file error: %v", err)
	}
	return err
}

func RemovePidFile() {
	if pidFile != "" {
		os.Remove(pidFile)
	}
}

// GetPidFrom gets pid from specified file_path
func GetPidFrom(pidFilePath string) (pid int, err error) {

	if pidFilePath == "" {
		pidFilePath = types.MosnPidDefaultFileName
	}

	var pf io.Reader
	if pf, err = os.Open(pidFilePath); err != nil {
		return
	}

	var bs []byte
	if bs, err = ioutil.ReadAll(pf); err != nil {
		return
	}

	pid, err = strconv.Atoi(strings.TrimRight(string(bs), "\n"))
	return
}
