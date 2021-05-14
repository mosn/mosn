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

package codec

import (
	"context"
	"errors"
	"net/http"

	"mosn.io/api"
)

type StatusMapping struct{}

func (m *StatusMapping) MappingHeaderStatusCode(ctx context.Context, headers api.HeaderMap) (int, error) {
	cmd, ok := headers.(api.XRespFrame)
	if !ok {
		return 0, errors.New("no response status in headers")
	}
	code := uint16(cmd.GetStatusCode())
	// TODO: more accurate mapping
	switch code {
	case ResponseStatusSuccess:
		return http.StatusOK, nil
	default:
		return http.StatusInternalServerError, nil
	}
}
