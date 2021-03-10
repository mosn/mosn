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
	"runtime/debug"
	"strings"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type MetadataFilter struct {
	metadataers *FilterConfigFactory
}

// NewMetadataFilter used to create new metadata filter
func NewMetadataFilter(f *FilterConfigFactory) *MetadataFilter {
	filter := &MetadataFilter{
		metadataers: f,
	}
	return filter
}

func (f *MetadataFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [metadata] OnReceive() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.metadataers.disable {
		return api.StreamFilterContinue
	}

	f.buildMetadater(ctx, headers, buf, trailers)

	return api.StreamFilterContinue
}

func (f *MetadataFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	return api.StreamFilterContinue
}

func (f *MetadataFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}

func (f *MetadataFilter) OnDestroy() {}

func (f *MetadataFilter) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {

}

func (f *MetadataFilter) buildMetadater(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) {
	for key, driver := range f.metadataers.drivers {
		if v, err := driver.BuildMetadata(ctx, headers, buf, trailers); err != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [metadata] buildMetadater() err: %v", err)
		} else {

			if f.metadataers.caseSensitive {
				headers.Set(metadataPrefix+key, v)
			} else {
				headers.Set(strings.ToLower(metadataPrefix+key), strings.ToLower(v))
			}
		}
	}
}
