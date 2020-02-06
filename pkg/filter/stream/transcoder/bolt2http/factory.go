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

package bolt2http

import (
	"context"

	"mosn.io/mosn/pkg/api/v2"
	"mosn.io/mosn/pkg/filter"
	"mosn.io/mosn/pkg/types"
)

func init() {
	filter.RegisterStream(v2.MosnBoltHttpTranscoder, createFilterChainFactory)
}

type filterChainFactory struct {
	cfg *config
}

func (f *filterChainFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	transcodeFilter := newTranscodeFilter(context, f.cfg)
	callbacks.AddStreamReceiverFilter(transcodeFilter, types.DownFilterAfterRoute)
	callbacks.AddStreamSenderFilter(transcodeFilter)
}

func createFilterChainFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	cfg, err := parseConfig(conf)
	if err != nil {
		return nil, err
	}
	return &filterChainFactory{cfg}, nil
}
