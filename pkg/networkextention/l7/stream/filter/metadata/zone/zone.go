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

package zone

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"mosn.io/api"
	"mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata"
	"mosn.io/pkg/buffer"
)

func init() {
	metadata.RegisterDriver("ZONE", &ZoneDriver{})
}

type ZoneConfig struct {
	ZoneEnvNameList []string `json:"zone_env_name_list,omitempty"`
}

type ZoneDriver struct {
	zoneName string
}

func (d *ZoneDriver) Init(cfg map[string]interface{}) error {
	ic := &ZoneConfig{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, ic); err != nil {
		return err
	}

	if len(ic.ZoneEnvNameList) == 0 {
		return errors.New("need set zone env list via zone_env_name_list config")
	}

	for _, name := range ic.ZoneEnvNameList {
		d.zoneName = os.Getenv(name)
		if d.zoneName != "" {
			return nil
		}
	}
	if d.zoneName == "" {
		return errors.New("get zone name is nil, zone env name: " + strings.Join(ic.ZoneEnvNameList, ","))

	}
	return nil
}

func (d *ZoneDriver) BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error) {
	return d.zoneName, nil
}
