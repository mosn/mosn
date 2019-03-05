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

import "os"

func init() {
	os.MkdirAll(MosnLogBasePath, 0644)
	os.MkdirAll(MosnConfigPath, 0644)
}

var (
	MosnBasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "mosn"

	MosnLogBasePath    = MosnBasePath + string(os.PathSeparator) + "logs"
	MosnLogDefaultPath = MosnLogBasePath + string(os.PathSeparator) + "mosn.log"

	MosnConfigPath = MosnBasePath + string(os.PathSeparator) + "conf"

	ReconfigureDomainSocket    = MosnConfigPath + string(os.PathSeparator) + "reconfig.sock"
	TransferListenDomainSocket = MosnConfigPath + string(os.PathSeparator) + "listen.sock"
	TransferStatsDomainSocket  = MosnConfigPath + string(os.PathSeparator) + "stats.sock"
	TransferConnDomainSocket   = MosnConfigPath + string(os.PathSeparator) + "conn.sock"

	MosnPidFileName        = "mosn.pid"
	MosnPidDefaultFileName = MosnLogBasePath + string(os.PathSeparator) + MosnPidFileName
)
