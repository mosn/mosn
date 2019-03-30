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
	"strconv"
	"time"

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/log"
)

var bitSize64 = 1 << 6

func parseProxyTimeout(route types.Route, headers types.HeaderMap) *Timeout {
	timeout := &Timeout{}
	timeout.GlobalTimeout = route.RouteRule().GlobalTimeout()
	timeout.TryTimeout = route.RouteRule().Policy().RetryPolicy().TryTimeout()

	// todo: check global timeout in request headers
	// todo: check per try timeout in request headers

	if tto, ok := headers.Get(types.HeaderTryTimeout); ok {
		if trytimeout, err := strconv.ParseInt(tto, 10, bitSize64); err == nil {
			timeout.TryTimeout = time.Duration(trytimeout) * time.Millisecond
		}
	}

	if gto, ok := headers.Get(types.HeaderGlobalTimeout); ok {
		if globaltimeout, err := strconv.ParseInt(gto, 10, bitSize64); err == nil {
			timeout.GlobalTimeout = time.Duration(globaltimeout) * time.Millisecond
		}
	}

	if timeout.TryTimeout >= timeout.GlobalTimeout {
		timeout.TryTimeout = 0
	}

	log.DefaultLogger.Debugf("trytimeout %v, globaltimeout %v", timeout.TryTimeout, timeout.GlobalTimeout)

	return timeout
}
