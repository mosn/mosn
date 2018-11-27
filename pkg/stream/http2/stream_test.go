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
	"net/url"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

func Test_clientStream_AppendHeaders(t *testing.T) {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")
	URL, _ := url.Parse("http://127.0.0.1:12200")

	clientStreamsMocked := []clientStream{
		{
			stream: stream{
				request: new(http.Request),
			},
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					connection: network.NewClientConnection(nil, nil, remoteAddr, nil, log.DefaultLogger),
				},
			},
		},
		{
			stream: stream{
				request: &http.Request{
					URL: URL,
				},
			},
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					connection: network.NewClientConnection(nil, nil, remoteAddr, nil, log.DefaultLogger),
				},
			},
		},
		{
			stream: stream{
				request: new(http.Request),
			},
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
		{
			protocol.MosnHeaderQueryStringKey: "",
			protocol.MosnHeaderPathKey:        path,
		},
	}

	wantedURL := []string{
		"name=biz&passwd=bar",
		"",
		"",
	}

	for i := 0; i < len(clientStreamsMocked); i++ {
		clientStreamsMocked[i].AppendHeaders(nil, headers[i], false)
		if len(headers[i]) != 0 || clientStreamsMocked[i].request.URL.RawQuery != wantedURL[i] {
			t.Errorf("clientStream AppendHeaders() error, num: %d,  actual RawQuery: %s, expect Query: %s",
				i, clientStreamsMocked[i].request.URL.RawQuery, wantedURL[i])
		}
		if clientStreamsMocked[i].request.URL.ForceQuery {
			t.Errorf("ForceQuery should be false")
		}
	}
}
