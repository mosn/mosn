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

package idc

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
	metadata.RegisterDriver("IDC", &IdcDriver{})
}

type IdcConfig struct {
	IdcEnvNameList []string `json:"idc_env_name_list,omitempty"`
}

type IdcDriver struct {
	idcName string
}

func (d *IdcDriver) Init(cfg map[string]interface{}) error {
	ic := &IdcConfig{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, ic); err != nil {
		return err
	}

	if len(ic.IdcEnvNameList) == 0 {
		return errors.New("need set idc env list via idc_env_name_list config")
	}

	for _, name := range ic.IdcEnvNameList {
		d.idcName = os.Getenv(name)
		if d.idcName != "" {
			return nil
		}
	}
	if d.idcName == "" {
		return errors.New("get idc name is nil, idc env name: " + strings.Join(ic.IdcEnvNameList, ","))

	}
	return nil
}

func (d *IdcDriver) BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error) {
	return d.idcName, nil
}
