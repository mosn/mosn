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

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

const RequestTranscodeFail api.ResponseFlag = 0x2000

// Transcoder provide ability to transcoding request/response
type Transcoder interface {
	// Accept
	Accept(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) bool
	// TranscodingRequest
	TranscodingRequest(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error)
	// TranscodingResponse
	TranscodingResponse(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) (types.HeaderMap, types.IoBuffer, types.HeaderMap, error)
}
