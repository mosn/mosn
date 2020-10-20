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

package mtls

import (
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

var (
	getSdsClientFunc           func(cfg *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig) types.SdsClient = sds.NewSdsClientSingleton
	getSdsClientFuncDeprecated func(cfg *envoy_api_v2_auth.SdsSecretConfig) types.SdsClientDeprecated               = sds.NewSdsClientSingletonDeprecated
)

func GetSdsClient(cfg *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig) types.SdsClient {
	return getSdsClientFunc(cfg)
}

func GetSdsClientDeprecated(cfg *envoy_api_v2_auth.SdsSecretConfig) types.SdsClientDeprecated {
	return getSdsClientFuncDeprecated(cfg)
}
