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

package gzip

import (
	"context"
	"encoding/json"
	"errors"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

func init() {
	api.RegisterStream(v2.Gzip, CreateGzipFilterFactory)
}

type FilterConfigFactory struct {
	Config *v2.StreamGzip
}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewStreamFilter(context, f.Config)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

func CreateGzipFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	log.DefaultLogger.Debugf("create gzip stream filter factory")
	cfg, err := ParseStreamGzipFilter(conf)
	if err != nil {
		return nil, err
	}

	if err = checkValidConfig(cfg); err != nil {
		return nil, err
	}
	return &FilterConfigFactory{cfg}, nil
}

// ParseStreamGzipFilter
func ParseStreamGzipFilter(cfg map[string]interface{}) (*v2.StreamGzip, error) {
	filterConfig := &v2.StreamGzip{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return filterConfig, nil
}

func checkValidConfig(cfg *v2.StreamGzip) error {
	if cfg.GzipLevel < 0 || cfg.GzipLevel > 9 {
		return errors.New("invalid gzip level, the values are in the range from 0 to 9.")
	}

	if cfg.GzipLevel == 0 {
		cfg.GzipLevel = defaultGzipLevel
	}

	if cfg.ContentType == nil {
		cfg.ContentType = []string{defaultContentType}
	}

	return nil
}
