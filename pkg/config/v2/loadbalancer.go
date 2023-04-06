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

package v2

import (
	"mosn.io/api"
)

type LeastRequestLbConfig struct {
	ChoiceCount       uint32
	ActiveRequestBias float64 `json:"active_request_bias,omitempty"`
}

func (lbconfig *LeastRequestLbConfig) isCluster_LbConfig() {
}

type IsCluster_LbConfig interface {
	isCluster_LbConfig()
}

type HashPolicy struct {
	Header   *HeaderHashPolicy   `json:"header,omitempty"`
	Cookie   *CookieHashPolicy   `json:"cookie,omitempty"`
	SourceIP *SourceIPHashPolicy `json:"source_ip,omitempty"`
}

type HeaderHashPolicy struct {
	Key string `json:"key,omitempty"`
}

type CookieHashPolicy struct {
	Name string             `json:"name,omitempty"`
	Path string             `json:"path,omitempty"`
	TTL  api.DurationConfig `json:"ttl,omitempty"`
}

type SourceIPHashPolicy struct {
}
