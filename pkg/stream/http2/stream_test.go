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

package http2

import (
	"net"
	"net/http"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

func Test_clientStream_AppendHeaders(t *testing.T) {
	streamMocked := stream{
		request: new(http.Request),
	}
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	clientStreamsMocked := []clientStream{
		{
			stream: streamMocked,
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					connection: network.NewClientConnection(nil, nil, remoteAddr, nil, log.DefaultLogger),
				},
			},
		},
		{
			stream: streamMocked,
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					connection: network.NewClientConnection(nil, nil, remoteAddr, nil, log.DefaultLogger),
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			protocol.MosnHeaderQueryStringKey: queryString,
			protocol.MosnHeaderPathKey:        path,
		},
		{
			protocol.MosnHeaderQueryStringKey: queryString,
		},
	}

	wantedURL := []string{
		"name=biz&passwd=bar",
		"",
	}

	for i := 0; i < len(clientStreamsMocked); i++ {
		clientStreamsMocked[i].AppendHeaders(nil, headers[i], false)
		if len(headers[i]) != 0 && clientStreamsMocked[i].request.URL.RawQuery != wantedURL[i] {
			t.Errorf("clientStream AppendHeaders() error")
		}
	}

}
