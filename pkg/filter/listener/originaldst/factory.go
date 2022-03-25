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
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

// OriginDST filter used to find out destination address of a connection which been redirected by iptables or user header.
func init() {
	api.RegisterListener(v2.ORIGINALDST_LISTENER_FILTER, CreateOriginalDstFactory)
}

type OriginalDstConfig struct {
	// If FallbackToLocal is setted to true, the listener filter match will use local address instead of
	// any (0.0.0.0). usually used in ingress listener.
	FallbackToLocal bool `json:"fallback_to_local"`
}

type originalDst struct {
	fallbackToLocal bool
}

// TODO remove it when Istio deprecate UseOriginalDst.
// NewOriginalDst new an original dst filter.
func NewOriginalDst() api.ListenerFilterChainFactory {
	return &originalDst{}
}

func CreateOriginalDstFactory(conf map[string]interface{}) (api.ListenerFilterChainFactory, error) {
	b, _ := json.Marshal(conf)
	cfg := OriginalDstConfig{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &originalDst{
		fallbackToLocal: cfg.FallbackToLocal,
	}, nil
}

const localHost = "127.0.0.1"

// OnAccept called when connection accept
func (filter *originalDst) OnAccept(cb api.ListenerFilterChainFactoryCallbacks) api.FilterStatus {
	if !cb.GetUseOriginalDst() {
		return api.Continue
	}

	ip, port, err := getOriginalAddr(cb.Conn())
	if err != nil {
		log.DefaultLogger.Errorf("[originaldst] get original addr failed: %v", err)
		return api.Continue
	}

	ips := fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("originalDst remote addr: %s:%d", ips, port)
	}
	if filter.fallbackToLocal {
		ctx := cb.GetOriContext()
		variable.SetString(ctx, types.VarListenerMatchFallbackIP, localHost)
	}

	cb.SetOriginalAddr(ips, port)
	cb.UseOriginalDst(cb.GetOriContext())

	return api.Stop
}
