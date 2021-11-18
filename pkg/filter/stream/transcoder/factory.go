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

package transcoder

import (
	"context"
	"mosn.io/api/extensions/transcoder"
	v2 "mosn.io/mosn/pkg/config/v2"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// stream factory
func init() {
	api.RegisterStream(v2.Transcoder, createFilterChainFactory)
}

type filterChainFactory struct {
	cfg *config
}

func (f *filterChainFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	transcodeFilter := newTranscodeFilter(context, f.cfg)
	if transcodeFilter != nil {
		callbacks.AddStreamReceiverFilter(transcodeFilter, api.AfterRoute)
		callbacks.AddStreamSenderFilter(transcodeFilter, api.BeforeSend)
	}
}

func createFilterChainFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	cfg, err := parseConfig(conf)
	if err != nil {
		return nil, err
	}
	return &filterChainFactory{cfg}, nil
}

// transcoder factory
var transcoderFactory = make(map[string]transcoder.Transcoder)

func MustRegister(typ string, transcoder transcoder.Transcoder) {
	if transcoderFactory[typ] != nil {
		panic("target stream transcoder already exists: " + typ)
	}

	transcoderFactory[typ] = transcoder
}

func GetTranscoder(typ string) transcoder.Transcoder {

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream filter][transcoder] GetTranscoder, typ %s, transcoderFactory %+v", typ, transcoderFactory)
	}

	return transcoderFactory[typ]
}
