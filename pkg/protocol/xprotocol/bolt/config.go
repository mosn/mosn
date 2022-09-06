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

package bolt

import (
	"context"
	"encoding/json"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func init() {
	protocol.RegisterProtocolConfigHandler(ProtocolName, ConfigHandler)
}

type Config struct {
	EnableBoltGoAway bool `json:"enable_bolt_goaway,omitempty"`
}

var defaultConfig = Config{
	EnableBoltGoAway: false,
}

func ConfigHandler(v interface{}) interface{} {
	extendConfig, ok := v.(map[string]interface{})
	if !ok {
		return defaultConfig
	}

	tmpConfig, ok := extendConfig[string(ProtocolName)]
	if !ok {
		tmpConfig = extendConfig
	}
	tmpConfigBytes, err := json.Marshal(tmpConfig)
	if err != nil {
		return defaultConfig
	}
	config := defaultConfig
	if err := json.Unmarshal(tmpConfigBytes, &config); err != nil {
		return defaultConfig
	}

	return config
}

func parseConfig(ctx context.Context) Config {
	config := defaultConfig
	// get extend config from ctx
	if pgc, err := variable.Get(ctx, types.VariableProxyGeneralConfig); err == nil && pgc != nil {
		if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
			if boltConfig, ok := extendConfig[ProtocolName]; ok {
				if cfg, ok := boltConfig.(Config); ok {
					config = cfg
				}
			}
		}
	}
	return config
}
