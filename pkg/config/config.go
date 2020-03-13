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

package config

import (
	v2 "sofastack.io/sofa-mosn/pkg/api/v2"
)

type ContentKey string

// 使用alias机制，减少适配复杂度
type MOSNConfig = v2.MOSNConfig
type PProfConfig = v2.PProfConfig
type ClusterManagerConfigJson = v2.ClusterManagerConfigJson
type ClusterManagerConfig = v2.ClusterManagerConfig
type MetricsConfig = v2.MetricsConfig
type TracingConfig = v2.TracingConfig

// protetced configPath, read only
func GetConfigPath() string {
	return configPath
}
