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
	"encoding/json"
	"errors"
)

type ParseExtendConfig func(config json.RawMessage) error

var extendConfigs map[string]ParseExtendConfig

func init() {
	extendConfigs = map[string]ParseExtendConfig{}
}

var ErrDuplicateExtendConfigParse = errors.New("duplicate parse functions of the type")

// RegisterParseExtendConfig should be called before config parse.
func RegisterParseExtendConfig(typ string, f ParseExtendConfig) error {
	if _, ok := extendConfigs[typ]; ok {
		return ErrDuplicateExtendConfigParse
	}
	extendConfigs[typ] = f
	return nil
}

// ExtendConfigParsed called the registed ParseExtendConfig
// Notice the ParseExtendConfig maybe makes the config parse slowly.
func ExtendConfigParsed(typ string, cfg json.RawMessage) error {
	if f, ok := extendConfigs[typ]; ok {
		return f(cfg)
	}
	// extend config type is not registered, ignore
	return nil
}
