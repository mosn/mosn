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

package ipaccess

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/log"
	"mosn.io/pkg/variable"
)

type IPAccessFilter struct {
	access   []IPAccess
	header   string
	allowAll bool // use by meet a block ip
	handler  api.StreamReceiverFilterHandler
}

func NewIPAccessFilter(access []IPAccess, header string, allowAll bool) *IPAccessFilter {
	return &IPAccessFilter{
		access:   access,
		header:   header,
		allowAll: allowAll,
	}
}
func (f *IPAccessFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	var addr string
	if f.header != "" {
		addr, _ = headers.Get(f.header)
	}
	if addr == "" {
		if cv, err := variable.Get(ctx, types.VariableConnection); err == nil {
			if conn, ok := cv.(api.Connection); ok {
				addr = conn.RemoteAddr().String()
			}
		}
	}

	// match all ip access logic
	for _, v := range f.access {
		allow, err := v.IsAllow(addr)
		if err != nil {
			log.DefaultLogger.Infof("%s is a broken ip with error :%v ", addr, err)
			continue
		}
		// allowlist should  continue
		if allow && !v.IsDenyAction() {
			return api.StreamFilterContinue
		}

		// blocklist should stop now
		if !allow && v.IsDenyAction() {
			f.handler.SendHijackReply(403, headers)
			log.DefaultLogger.Infof("%s has been blocked by ip_access", addr)
			return api.StreamFilterStop
		}
	}
	if f.allowAll {
		return api.StreamFilterContinue
	}
	f.handler.SendHijackReply(403, headers)
	log.DefaultLogger.Infof("%s has been blocked by ip_access", addr)
	return api.StreamFilterStop
}

func (f *IPAccessFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *IPAccessFilter) OnDestroy() {}
