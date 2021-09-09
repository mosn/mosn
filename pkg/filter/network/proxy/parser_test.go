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

package proxy

import (
	"encoding/json"
	"testing"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

func TestParseProxyFilter(t *testing.T) {
	proxyConfigStr := `{
                "name": "proxy",
                "downstream_protocol": "X",
                "upstream_protocol": "Http2",
                "router_config_name":"test_router",
                "extend_config":{
                        "sub_protocol":"example"
                }
        }`
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(proxyConfigStr), &m); err != nil {
		t.Error(err)
		return
	}
	proxy, _ := ParseProxyFilter(m)
	if !(proxy.Name == "proxy" &&
		proxy.RouterHandlerName == types.DefaultRouteHandler &&
		proxy.DownstreamProtocol == string(protocol.Xprotocol) &&
		proxy.UpstreamProtocol == string(protocol.HTTP2) && proxy.RouterConfigName == "test_router") {
		t.Error("parse proxy filter failed")
	}
}
