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

package bolt

import (
	"context"
	"errors"
	"net/http"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMapping(ProtocolName, &boltStatusMapping{})
}

type boltStatusMapping struct{}

func (m *boltStatusMapping) MappingHeaderStatusCode(ctx context.Context, headers types.HeaderMap) (int, error) {
	cmd, ok := headers.(xprotocol.XRespFrame)
	if !ok {
		return 0, errors.New("no response status in headers")
	}
	code := uint16(cmd.GetStatusCode())
	// TODO: more accurate mapping
	switch code {
	case ResponseStatusSuccess:
		return http.StatusOK, nil
	case ResponseStatusServerThreadpoolBusy:
		return http.StatusServiceUnavailable, nil
	case ResponseStatusTimeout:
		return http.StatusGatewayTimeout, nil
		//case RESPONSE_STATUS_CLIENT_SEND_ERROR: // CLIENT_SEND_ERROR maybe triggered by network problem, 404 is not match
		//	return http.StatusNotFound, nil
	case ResponseStatusConnectionClosed:
		return http.StatusBadGateway, nil
	default:
		return http.StatusInternalServerError, nil
	}
}
