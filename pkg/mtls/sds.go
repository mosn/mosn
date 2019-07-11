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
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"sofastack.io/sofa-mosn/pkg/types"
)

// TODO: move into types
type SdsClient interface {
	AddUpdateCallback(sdsConfig *auth.SdsSecretConfig, f func(string, *types.SDSSecret))
	SetSecret(name string, secret *types.SDSSecret)
}

// TODO: use v2 config as default
var getSdsClientFunc func(cfg *auth.SdsSecretConfig) SdsClient

func GetSdsClient(cfg *auth.SdsSecretConfig) SdsClient {
	return getSdsClientFunc(cfg)
}
