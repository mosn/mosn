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
	"path/filepath"
	"strings"
)

var (
	MosnBasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "mosn"

	MosnLogBasePath        = MosnBasePath + string(os.PathSeparator) + "logs"
	MosnLogDefaultPath     = MosnLogBasePath + string(os.PathSeparator) + "mosn.log"
	MosnLogProxyPath       = MosnLogBasePath + string(os.PathSeparator) + "proxy.log"
	MosnPidDefaultFileName = MosnLogBasePath + string(os.PathSeparator) + "mosn.pid"

	MosnConfigPath = MosnBasePath + string(os.PathSeparator) + "conf"

	ReconfigureDomainSocket    = MosnConfigPath + string(os.PathSeparator) + "reconfig.sock"
	TransferConnDomainSocket   = MosnConfigPath + string(os.PathSeparator) + "conn.sock"
	TransferStatsDomainSocket  = MosnConfigPath + string(os.PathSeparator) + "stats.sock"
	TransferListenDomainSocket = MosnConfigPath + string(os.PathSeparator) + "listen.sock"
)

func InitDefaultPath(path string) {
	var err error
	var index int
	var config string

	if path == "" {
		goto end
	}
	config, err = filepath.Abs(filepath.Dir(path))
	if err != nil {
		goto end
	}
	index = strings.LastIndex(config, string(os.PathSeparator))
	if index == -1 || index == 0 {
		goto end
	}

	MosnBasePath = config[0:index]

	MosnLogBasePath = MosnBasePath + string(os.PathSeparator) + "logs"

	MosnLogDefaultPath = MosnLogBasePath + string(os.PathSeparator) + "mosn.log"
	MosnPidDefaultFileName = MosnLogBasePath + string(os.PathSeparator) + "mosn.pid"

	MosnConfigPath = config

	ReconfigureDomainSocket = MosnConfigPath + string(os.PathSeparator) + "reconfig.sock"
	TransferConnDomainSocket = MosnConfigPath + string(os.PathSeparator) + "conn.sock"
	TransferStatsDomainSocket = MosnConfigPath + string(os.PathSeparator) + "stats.sock"
	TransferListenDomainSocket = MosnConfigPath + string(os.PathSeparator) + "listen.sock"

end:
	os.MkdirAll(MosnLogBasePath, 0755)
	os.MkdirAll(MosnConfigPath, 0755)
}
