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
	"strings"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func init() {
	api.RegisterNetwork(v2.DEFAULT_NETWORK_FILTER, CreateProxyFactory)
}

type genericProxyFilterConfigFactory struct {
	Proxy *v2.Proxy
	//
	extendConfig map[api.ProtocolName]interface{}
	protocols    []api.ProtocolName
}

func (gfcf *genericProxyFilterConfigFactory) CreateFilterChain(ctx context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	if gfcf.extendConfig != nil {
		_ = variable.Set(ctx, types.VariableProxyGeneralConfig, gfcf.extendConfig)
	}

	// TODO: cache it, use varProtocolConfig instead search with Set API
	variable.Set(ctx, types.VarProtocolConfig, gfcf.protocols)

	p := proxy.NewProxy(ctx, gfcf.Proxy)
	callbacks.AddReadFilter(p)
}

var protocolCheck func(api.ProtocolName) bool = protocol.ProtocolRegistered

func CreateProxyFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	p, err := ParseProxyFilter(conf)
	if err != nil {
		return nil, err
	}
	gfcf := &genericProxyFilterConfigFactory{
		Proxy: p,
	}

	protos := strings.Split(p.DownstreamProtocol, ",")
	gfcf.protocols = make([]api.ProtocolName, 0, len(protos))
	for _, p := range protos {
		proto := api.ProtocolName(p)
		if ok := protocolCheck(proto); !ok {
			return nil, fmt.Errorf("invalid downstream protocol %s", p)
		}
		gfcf.protocols = append(gfcf.protocols, proto)
	}

	if len(p.ExtendConfig) != 0 {
		gfcf.extendConfig = make(map[api.ProtocolName]interface{})
		// It's just for backward compatibility, will be removed in the feature.
		if len(gfcf.protocols) == 1 && gfcf.protocols[0] != protocol.Auto {
			proto := gfcf.protocols[0]
			gfcf.extendConfig[proto] = protocol.HandleConfig(proto, p.ExtendConfig)
			if p.ExtendConfig[string(proto)] == nil {
				log.DefaultLogger.Warnf(`Old extend_config format is deprecated, please use the protocol name as key index, eg: "Http2: {http2_use_stream: true}"`)
			}
		} else {
			for proto, value := range p.ExtendConfig {
				gfcf.extendConfig[api.ProtocolName(proto)] = protocol.HandleConfig(api.ProtocolName(proto), value)
			}
		}
	}
	return gfcf, nil
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
	if proxyConfig.DownstreamProtocol == "" {
		return nil, fmt.Errorf("protocol in string needed in proxy network filter")
	}
	// TODO: compatbile handler, will be removed later.
	if proxyConfig.DownstreamProtocol == "X" && len(proxyConfig.ExtendConfig) != 0 {
		if v, ok := proxyConfig.ExtendConfig["sub_protocol"]; ok {
			if subProtocol, ok := v.(string); ok {
				proxyConfig.DownstreamProtocol = subProtocol
			}
		}
	}
	// set default proxy router name
	if proxyConfig.RouterHandlerName == "" {
		proxyConfig.RouterHandlerName = router.GetDefaultRouteHandlerName()
	}
	if !router.MakeHandlerFuncExists(proxyConfig.RouterHandlerName) {
		log.DefaultLogger.Alertf(types.ErrorKeyConfigParse, "proxy router handler is not exists, will use default handler instead")
	}

	return proxyConfig, nil
}
