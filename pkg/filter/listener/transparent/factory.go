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

package transparent

import (
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

func init() {
	api.RegisterListener(v2.TRANSPARENT_LISTENER_FILTER, CreateTransparentProxyFactory)
}

type transparentProxy struct {
}

func NewTransparentProxy() api.ListenerFilterChainFactory {
	return &transparentProxy{}
}

func CreateTransparentProxyFactory(conf map[string]interface{}) (api.ListenerFilterChainFactory, error) {
	return &transparentProxy{}, nil
}

// OnAccept called when connection accept
func (filter *transparentProxy) OnAccept(cb api.ListenerFilterChainFactoryCallbacks) api.FilterStatus {
	if !cb.GetUseOriginalDst() {
		return api.Continue
	}

	ip, port, err := getOriginalAddr(cb.Conn())
	if err != nil {
		log.DefaultLogger.Errorf("[transparentProxy] get original addr failed: %v", err)
		return api.Continue
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("transparentProxy remote addr: %s:%d", ip, port)
	}

	cb.SetOriginalAddr(ip, port)
	cb.UseOriginalDst(cb.GetOriContext())

	return api.Stop
}
