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

import "sofastack.io/sofa-mosn/common/log"

// module name
const ErrorModuleMosn log.ErrorKey = "sofa-mosn."

// sub module name
const (
	ErrorSubModuleIO     log.ErrorKey = "io."
	ErrorSubModuleProxy               = "proxy."
	ErrorSubModuleAdmin               = "admin."
	ErrorSubModuleCommon              = "common."
)

// error keys
const (
	ErrorKeyAdmin        log.ErrorKey = ErrorModuleMosn + ErrorSubModuleAdmin + "admin_failed"
	ErrorKeyConfigDump                = ErrorModuleMosn + ErrorSubModuleCommon + "config_dump_failed"
	ErrorKeyReconfigure               = ErrorModuleMosn + ErrorSubModuleCommon + "reconfigure_failed"
	ErrorKeyTLSFallback               = ErrorModuleMosn + ErrorSubModuleCommon + "tls_fallback"
	ErrorKeyAppendHeader              = ErrorModuleMosn + ErrorSubModuleProxy + "append_header_failed"
	ErrorKeyRouteMatch                = ErrorModuleMosn + ErrorSubModuleProxy + "route_match_failed"
	ErrorKeyClusterGet                = ErrorModuleMosn + ErrorSubModuleProxy + "cluster_get_failed"
	ErrorKeyUpstreamConn              = ErrorModuleMosn + ErrorSubModuleProxy + "upstream_conn_failed"
	ErrorKeyCodec                     = ErrorModuleMosn + ErrorSubModuleProxy + "codec_error"
	ErrorKeyHeartBeat                 = ErrorModuleMosn + ErrorSubModuleProxy + "heartbeat_unknown"
	// TODO: more keys
)
