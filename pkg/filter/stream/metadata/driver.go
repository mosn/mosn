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

package metadata

import (
	"context"
	"errors"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

const (
	metadataPrefix = "x-mosn-on-envoy-"
)

var (
	drivers = make(map[string]Driver)
)

type Driver interface {
	Init(config map[string]interface{}) error
	BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error)
}

func RegisterDriver(typ string, driver Driver) {
	drivers[typ] = driver
}

func InitMetadataDriver(typ string, config map[string]interface{}) (Driver, error) {
	if driver, ok := drivers[typ]; ok {
		if err := driver.Init(config); err != nil {
			return nil, err
		} else {
			return driver, nil
		}
	}
	return nil, errors.New("not found dirver: " + typ)
}
