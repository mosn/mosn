//go:build linux || darwin
// +build linux darwin

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

package types

import (
	"os"
	"path"
	"testing"
)

func TestInitDefaultPath(t *testing.T) {
	// init
	testPath := "/tmp/mosn_default/conf"
	os.RemoveAll(testPath)
	testConfigPath := path.Join(testPath, "testfile.json")
	// test
	InitDefaultPath(testConfigPath, "")
	// verify
	// if config is /tmp/mosn_defaulta/conf/config.json
	// the log should in /tmp/mosn_default/logs/*
	// the others shoykd in  /tmp/mosn_default/conf*
	if !(MosnLogBasePath == path.Join("/tmp/mosn_default", "logs") &&
		MosnConfigPath == path.Join(testPath)) {
		t.Errorf("init default path failed: %s, %s", MosnLogBasePath, MosnConfigPath)
	}
	// invalid config should not changed the value
	InitDefaultPath("", "")
	if !(MosnLogBasePath == path.Join("/tmp/mosn_default", "logs") &&
		MosnConfigPath == path.Join(testPath)) {
		t.Errorf("init default path failed: %s, %s", MosnLogBasePath, MosnConfigPath)
	}
	InitDefaultPath("/tmp", "")
	if !(MosnLogBasePath == path.Join("/tmp/mosn_default", "logs") &&
		MosnConfigPath == path.Join(testPath)) {
		t.Errorf("init default path failed: %s, %s", MosnLogBasePath, MosnConfigPath)
	}
	// clean
	os.RemoveAll(testPath)
}

func TestInitUDSDir(t *testing.T) {
	ReconfigureDomainSocket = "/home/admin/mosn/conf/reconfig.sock"

	testCases := []struct {
		name         string
		UDSDir       string
		expectedPath string
	}{
		{
			name:         "empty_dir",
			UDSDir:       "",
			expectedPath: "/tmp/mosn/conf/reconfig.sock",
		},
		{
			name:         "normal_dir",
			UDSDir:       "/tmp/mosn/socks",
			expectedPath: "/tmp/mosn/socks/reconfig.sock",
		},
		{
			name:         "multiple_separator",
			UDSDir:       "/tmp//mosn/sock//",
			expectedPath: "/tmp/mosn/sock/reconfig.sock",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			InitDefaultPath("/tmp/mosn/conf/", testCase.UDSDir)
			if ReconfigureDomainSocket != testCase.expectedPath {
				t.Errorf("expected path: %s, got: %s", testCase.expectedPath, ReconfigureDomainSocket)
			}
		})
	}
}
