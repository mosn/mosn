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

package zipkin

import (
	mosnhttp "mosn.io/pkg/protocol/http"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

// extractHTTP will extract a span.Context from the RequestHeader if found in
// B3 header format.
func extractHTTP(r mosnhttp.RequestHeader) propagation.Extractor {
	return func() (*model.SpanContext, error) {
		singleHeader, ok := r.Get(b3.Context)
		if ok {
			ctx, err := b3.ParseSingleHeader(singleHeader)
			if err == nil {
				return ctx, nil
			} else {
				return nil, err
			}
		}

		var (
			hdrTraceID, _      = r.Get(b3.TraceID)
			hdrSpanID, _       = r.Get(b3.SpanID)
			hdrParentSpanID, _ = r.Get(b3.ParentSpanID)
			hdrSampled, _      = r.Get(b3.Sampled)
			hdrFlgs, _         = r.Get(b3.Flags)
		)

		ctx, err := b3.ParseHeaders(hdrTraceID, hdrSpanID, hdrParentSpanID, hdrSampled, hdrFlgs)
		if err != nil {
			return nil, err
		}
		return ctx, nil
	}
}
