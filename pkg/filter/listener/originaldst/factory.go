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

package originaldst

import (
	"encoding/json"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

const (
	TProxy   = "tproxy"
	Redirect = "redirect"
)

// OriginDST filter used to find out destination address of a connection which been redirected by iptables or user header.
func init() {
	api.RegisterListener(v2.ORIGINALDST_LISTENER_FILTER, CreateOriginalDstFactory)

}

type OriginalDstConfig struct {
	// If FallbackToLocal is setted to true, the listener filter match will use local address instead of
	// any (0.0.0.0). usually used in ingress listener.
	FallbackToLocal bool   `json:"fallback_to_local"`
	Type            string `json:"type"`
}

func CreateOriginalDstConfig(conf map[string]interface{}) (OriginalDstConfig, error) {
	b, _ := json.Marshal(conf)
	cfg := OriginalDstConfig{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Type == "" {
		cfg.Type = Redirect
	}

	return cfg, nil
}

type originalDst struct {
	FallbackToLocal bool
	Type            string
}

// TODO remove it when Istio deprecate UseOriginalDst.
// NewOriginalDst new an original dst filter.
func NewOriginalDst() api.ListenerFilterChainFactory {
	return &originalDst{}
}

func CreateOriginalDstFactory(conf map[string]interface{}) (api.ListenerFilterChainFactory, error) {
	cfg, err := CreateOriginalDstConfig(conf)
	if err != nil {
		return nil, err
	}
	return &originalDst{
		FallbackToLocal: cfg.FallbackToLocal,
		Type:            cfg.Type,
	}, nil
}

const localHost = "127.0.0.1"

// OnAccept called when connection accept
func (filter *originalDst) OnAccept(cb api.ListenerFilterChainFactoryCallbacks) api.FilterStatus {
	if !cb.GetUseOriginalDst() {
		return api.Continue
	}

	var ip string
	var port int
	var err error
	var logTag string

	if filter.Type == TProxy {
		ip, port, err = getTProxyAddr(cb.Conn())
		logTag = TProxy

	} else if filter.Type == Redirect {
		ip, port, err = getRedirectAddr(cb.Conn())
		logTag = Redirect
	} else {
		log.DefaultLogger.Errorf("listenerFifter type error: not %d or %d", TProxy, Redirect)
	}

	if err != nil {
		log.DefaultLogger.Errorf("[%s] get original addr failed: %v", logTag, err)
		return api.Continue
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("%s remote addr: %s:%d", logTag, ip, port)
	}

	if filter.FallbackToLocal {
		ctx := cb.GetOriContext()
		variable.SetString(ctx, types.VarListenerMatchFallbackIP, localHost)
	}

	cb.SetOriginalAddr(ip, port)
	cb.UseOriginalDst(cb.GetOriContext())

	return api.Stop
}
