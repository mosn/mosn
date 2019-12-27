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

package router

import (
	"fmt"

	"mosn.io/mosn/pkg/types"
)

type headerParser struct {
	headersToAdd    []*headerPair
	headersToRemove []*lowerCaseString
}

func (h *headerParser) evaluateHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	if h == nil {
		return
	}
	for _, toAdd := range h.headersToAdd {
		value := toAdd.headerFormatter.format(requestInfo)
		if v, ok := headers.Get(toAdd.headerName.Get()); ok && len(v) > 0 && toAdd.headerFormatter.append() {
			value = fmt.Sprintf("%s,%s", v, value)
		}
		headers.Set(toAdd.headerName.Get(), value)
	}

	for _, toRemove := range h.headersToRemove {
		headers.Del(toRemove.Get())
	}
}
