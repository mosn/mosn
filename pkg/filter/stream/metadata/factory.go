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
	"encoding/json"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func init() {
	api.RegisterStream(v2.Metadata, CreateMetadataFilterFactory)
}

// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	drivers map[string]Driver
	disable bool
}

// CreateFilterChain for create metadata filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewMetadataFilter(f)
	// Register the runtime hook for the Metadata
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

// CreateMetadataFilterFactory for create metadata filter factory
func CreateMetadataFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}

	cfg := v2.StreamMetadata{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	drivers := make(map[string]Driver)
	for _, v := range cfg.MetaDataers {
		if driver, err := InitMetadataDriver(v.MetaKey, v.Config); err != nil {
			return nil, err
		} else {
			drivers[v.MetaKey] = driver
		}
	}

	return &FilterConfigFactory{drivers: drivers, disable: cfg.Disable}, nil
}
