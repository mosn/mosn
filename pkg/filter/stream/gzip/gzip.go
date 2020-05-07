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

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
)

// gzipConfig is parsed from v2.StreamGzip
// TODO support minLength,ContentType,gziplevel etc.
type gzipConfig struct {
	enable string
}

func makegzipConfig(cfg *v2.StreamGzip) *gzipConfig {
	gzipSwitch := "off"
	if cfg.GzipLevel > 0 {
		gzipSwitch = "on"
	}

	gzipConfig := &gzipConfig{
		enable: gzipSwitch,
	}

	return gzipConfig
}

// streamGzipFilter is an implement of api.StreamReceiverFilter
type streamGzipFilter struct {
	config *gzipConfig
}

func NewStreamFilter(ctx context.Context, cfg *v2.StreamGzip) api.StreamSenderFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream filter] [gzip] create a new gzip filter")
	}
	gf := makegzipConfig(cfg)

	variable.SetVariableValue(ctx, types.VarProxyGzipSwitch, gf.enable)
	return &streamGzipFilter{
		config: gf,
	}
}

func (f *streamGzipFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

func (f *streamGzipFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream filter] [gzip] gzip filter do append headers")
	}
	return api.StreamFilterContinue
}

func (f *streamGzipFilter) OnDestroy() {
}
