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
	"context"
	"encoding/json"
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/proxy"
)

func init() {
	api.RegisterNetwork(v2.DEFAULT_NETWORK_FILTER, CreateProxyFactory)
}

type genericProxyFilterConfigFactory struct {
	Proxy *v2.Proxy
}

func (gfcf *genericProxyFilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	p := proxy.NewProxy(context, gfcf.Proxy)
	callbacks.AddReadFilter(p)
}

func CreateProxyFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	p, err := ParseProxyFilter(conf)
	if err != nil {
		return nil, err
	}
	return &genericProxyFilterConfigFactory{
		Proxy: p,
	}, nil
}

// ParseProxyFilter
func ParseProxyFilter(cfg map[string]interface{}) (*v2.Proxy, error) {
	proxyConfig := &v2.Proxy{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, proxyConfig); err != nil {
		return nil, err
	}

	if proxyConfig.DownstreamProtocol == "" || proxyConfig.UpstreamProtocol == "" {
		return nil, fmt.Errorf("protocol in string needed in proxy network filter")
	} else if _, ok := configmanager.ProtocolsSupported[proxyConfig.DownstreamProtocol]; !ok {
		return nil, fmt.Errorf("invalid downstream protocol %s", proxyConfig.DownstreamProtocol)
	} else if _, ok := configmanager.ProtocolsSupported[proxyConfig.UpstreamProtocol]; !ok {
		return nil, fmt.Errorf("invalid upstream protocol %s", proxyConfig.UpstreamProtocol)
	}

	return proxyConfig, nil
}
