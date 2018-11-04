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

package sofarpc

import (
	"errors"
	"net/http"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	protocol.RegisterMapping(protocol.SofaRPC, &sofaMapping{})
}

type sofaMapping struct{}

func (m *sofaMapping) MappingHeaderStatusCode(headers types.HeaderMap) (int, error) {
	cmd, ok := headers.(ProtoBasicCmd)
	if !ok {
		return 0, errors.New("headers is not a sofa header")
	}
	code := int16(cmd.GetRespStatus())
	// TODO: more accurate mapping
	switch code {
	case RESPONSE_STATUS_SUCCESS:
		return http.StatusOK, nil
	case RESPONSE_STATUS_SERVER_THREADPOOL_BUSY:
		return http.StatusServiceUnavailable, nil
	case RESPONSE_STATUS_TIMEOUT:
		return http.StatusGatewayTimeout, nil
	//case RESPONSE_STATUS_CLIENT_SEND_ERROR: // CLIENT_SEND_ERROR maybe triggered by network problem, 404 is not match
	//	return http.StatusNotFound, nil
	case RESPONSE_STATUS_CONNECTION_CLOSED:
		return http.StatusBadGateway, nil
	default:
		return http.StatusInternalServerError, nil
	}
}
