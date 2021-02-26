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

package echo

import (
	"context"
	"encoding/json"
	"errors"

	"mosn.io/api"
)

func init() {
	api.RegisterStream(Echo, CreateEchoFilterFactory)
}

// Stream Filter's Type
const (
	Echo = "echo"
)

// FilterConfigFactory ...
// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	Status  int               `json:"status"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"header_list"`
}

// CreateFilterChain for create echo filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewEchoFilter(f)
	// Register the runtime hook for the Echo
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

// CreateEchoFilterFactory for create echo filter factory
func CreateEchoFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}

	cfg := &FilterConfigFactory{
		Headers: make(map[string]string),
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	// check config
	if cfg.Status < 0 {
		return nil, errors.New("The status must >= 0.")
	}

	return cfg, nil
}
