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
	"errors"
	"os"

	"mosn.io/api"
)

// MetadataConfig is a config for metadata
type MetadataConfig struct {
	MetaKey LbMeta `json:"filter_metadata"`
}
type LbMeta struct {
	LbMetaKey map[string]interface{} `json:"mosn.lb"`
}

func metadataToConfig(md api.Metadata) *MetadataConfig {
	if len(md) == 0 {
		return nil
	}
	m := make(map[string]interface{}, len(md))
	for k, v := range md {
		m[k] = v
	}
	return &MetadataConfig{
		MetaKey: LbMeta{
			LbMetaKey: m,
		},
	}
}

func configToMetadata(cfg *MetadataConfig) api.Metadata {
	result := api.Metadata{}
	if cfg != nil {
		mosnLb := cfg.MetaKey.LbMetaKey
		for k, v := range mosnLb {
			if vs, ok := v.(string); ok {
				result[k] = vs
			}
		}
	}
	return result
}

var ErrDuplicateTLSConfig = errors.New("tls_context and tls_context_set can only exists one at the same time")

var ErrDuplicateStaticAndDynamic = errors.New("only one of static config or dynamic config should be exists")

const MaxFilePath = 128

const sep = string(os.PathSeparator)
