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

package tars

import (
	"github.com/TarsCloud/TarsGo/tars"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMatcher(ProtocolName, tarsMatcher)
}

// predicate dubbo header len and compare magic number
func tarsMatcher(data []byte) types.MatchResult {
	pkgLen, status := tars.TarsRequest(data)
	if pkgLen == 0 && status == tars.PACKAGE_LESS {
		return types.MatchAgain
	}
	if pkgLen == 0 && status == tars.PACKAGE_ERROR {
		return types.MatchFailed
	}
	if status == tars.PACKAGE_FULL {
		return types.MatchSuccess
	}
	return types.MatchFailed
}
