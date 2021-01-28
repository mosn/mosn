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

package protocol

import (
	"context"
	"errors"
	"strconv"

	"mosn.io/api"
	"mosn.io/api/types"
	"mosn.io/pkg/protocol"
	"mosn.io/pkg/variable"
)

// HTTP and HTTP2 does not need mapping
func init() {
	protocol.RegisterMapping(HTTP1, &httpMapping{})
	protocol.RegisterMapping(HTTP2, &httpMapping{})
}

// HTTP get status directly
type httpMapping struct{}

func (m *httpMapping) MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error) {
	status, err := variable.GetVariableValue(ctx, types.VarHeaderStatus)
	if err != nil {
		return 0, errors.New("headers have no status code")
	}
	return strconv.Atoi(status)
}
