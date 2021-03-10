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

package unit

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"mosn.io/api"
	"mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata"
	"mosn.io/pkg/buffer"
)

var errNotFoundUnitKey = errors.New("not found unit key")

const defaultUnitKey = "userid"

func init() {
	metadata.RegisterDriver("UNIT", &UnitDriver{})
}

type UnitConfig struct {
	UnitKey string `json:"unit_key,omitempty"`
}

type UnitDriver struct {
	unitKey string
}

func (d *UnitDriver) Init(cfg map[string]interface{}) error {
	uc := &UnitConfig{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, uc); err != nil {
		return err
	}

	d.unitKey = uc.UnitKey
	// default use defaultUnitKey
	if len(d.unitKey) == 0 {
		d.unitKey = defaultUnitKey
	}

	return nil
}

func (d *UnitDriver) BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error) {
	uid, ok := headers.Get(d.unitKey)
	if !ok || len(uid) == 0 {
		return "", errNotFoundUnitKey
	}

	if nuid, err := strconv.Atoi(uid); err == nil {
		return d.calculateMockRouter(nuid), nil
	} else {
		return "", err
	}
}

func (d *UnitDriver) calculateMockRouter(uid int) string {
	if uid >= 1 && uid <= 100 {
		return "s1"
	} else if uid >= 101 && uid <= 200 {
		return "s2"
	} else {
		return "s3"
	}
}
