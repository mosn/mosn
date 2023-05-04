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

package seata

import (
	"context"
	"encoding/json"

	"mosn.io/api"

	"mosn.io/mosn/pkg/log"
)

func init() {
	// static seata stream filter factory
	api.RegisterStream(SEATA, CreateFilterFactory)
}

type factory struct {
	Conf *Seata
}

func (f *factory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter, err := NewFilter(f.Conf)
	if err == nil {
		callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
		callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
	} else {
		log.DefaultLogger.Errorf("failed to init seata filter, err: %v", err)
	}
}

func CreateFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	config, err := parseConfig(conf)
	if err != nil {
		return nil, err
	}
	return &factory{config}, nil
}

// parseConfig
func parseConfig(cfg map[string]interface{}) (*Seata, error) {
	filterConfig := &Seata{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return filterConfig, nil
}
