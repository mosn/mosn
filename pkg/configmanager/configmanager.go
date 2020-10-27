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

package configmanager

import v2 "mosn.io/mosn/pkg/config/v2"

// Deprecated: these functions should not be called.
// we keep these functions just for compatiable.
// If dump config is needed, use featuregate: auto_config to dump automatically
func AddOrUpdateClusterConfig(_ []v2.Cluster) {
	setDump()
}

func RemoveClusterConfig(_ []string) {
	setDump()
}

func DeleteClusterHost(_ string, _ string) {
	setDump()
}

func AddOrUpdateClusterHost(_ string, _ v2.Host) {
	setDump()
}

func UpdateClusterManagerTLS(_ v2.TLSConfig) {
	setDump()
}

func AddClusterWithRouter(_ []v2.Cluster, _ *v2.RouterConfiguration) {
	setDump()
}

func AddOrUpdateRouterConfig(_ *v2.RouterConfiguration) {
	setDump()
}

func AddOrUpdateListener(_ *v2.Listener) {
	setDump()
}
