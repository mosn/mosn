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

package sds

import v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

// var getSdsClientFunc func(cfg *auth.SdsSecretConfig) types.SdsClient = sds.NewSdsClientSingleton

var globalSds struct {
	// getSdsStreamClient for extensions
	getSdsStreamClient func(config interface{}) (SdsStreamClient, error)

	getSdsNativeClient func(config interface{}) (v2.SecretDiscoveryServiceClient, error)
}

// func GetSdsClient(cfg *auth.SdsSecretConfig) types.SdsClient {}

func RegisterSdsStreamClientFactory(f func(config interface{}) (SdsStreamClient, error)) {
	globalSds.getSdsStreamClient = f
}

func RegisterSdsNativeClientFactory(f func(config interface{}) (v2.SecretDiscoveryServiceClient, error)) {
	globalSds.getSdsNativeClient = f
}

func GetSdsStreamClient(config interface{}) (SdsStreamClient, error) {
	return globalSds.getSdsStreamClient(config)
}

func GetSdsNativeClient(config interface{}) (v2.SecretDiscoveryServiceClient, error) {
	return globalSds.getSdsNativeClient(config)
}
