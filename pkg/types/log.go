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

type ErrorKey string

// module name
const ErrorModuleMosn ErrorKey = "sofa-mosn."

// sub module name
const (
	ErrorSubModuleIO            ErrorKey = "io."
	ErrorSubModuleProxy                  = "proxy."
	ErrorSubModuleDynamicConfig          = "dynamic-config."
	ErrorSubModuleAdmin                  = "admin."
	ErrorSubModuleCommon                 = "common."
)

// error keys
const (
	ErrorKeyAdmin         ErrorKey = ErrorModuleMosn + ErrorSubModuleAdmin + "admin_failed"
	ErrorKeyConfigDump             = ErrorModuleMosn + ErrorSubModuleCommon + "config_dump_failed"
	ErrorKeyReconfigure            = ErrorModuleMosn + ErrorSubModuleCommon + "reconfigure_failed"
	ErrorKeyTLSFallback            = ErrorModuleMosn + ErrorSubModuleCommon + "tls_fallback"
	ErrorKeyRouteUpdate            = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "route_update_failed"
	ErrorKeyRouteAppend            = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "route_append_failed"
	ErrorKeyRouteClean             = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "route_clean_failed"
	ErrorKeyClusterAdd             = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "cluster_add_failed"
	ErrorKeyClusterUpdate          = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "cluster_update_failed"
	ErrorKeyClusterDelete          = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "cluster_delete_failed"
	ErrorKeyHostsUpdate            = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "hosts_update_failed"
	ErrorKeyHostsAppend            = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "hosts_append_failed"
	ErrorKeyHostsDelete            = ErrorModuleMosn + ErrorSubModuleDynamicConfig + "hosts_delete_failed"
	ErrorKeyAppendHeader           = ErrorModuleMosn + ErrorSubModuleProxy + "append_header_failed"
	ErrorKeyRouteMatch             = ErrorModuleMosn + ErrorSubModuleProxy + "route_match_failed"
	ErrorKeyClusterGet             = ErrorModuleMosn + ErrorSubModuleProxy + "cluster_get_failed"
	ErrorKeyUpstreamConn           = ErrorModuleMosn + ErrorSubModuleProxy + "upstream_conn_failed"
	ErrorKeyStreamFilter           = ErrorModuleMosn + ErrorSubModuleProxy + "streamfilter_unexpected"
	ErrorKeyCodec                  = ErrorModuleMosn + ErrorSubModuleProxy + "codec_error"
	ErrorKeyLoadBalance            = ErrorModuleMosn + ErrorSubModuleProxy + "loadbalance_failed"
	ErrorKeyHeartBeat              = ErrorModuleMosn + ErrorSubModuleProxy + "heartbeat_unknown"
	// TODO: more keys
)
