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
	tarsprotocol "github.com/TarsCloud/TarsGo/tars/protocol"
	"mosn.io/api"
)

// predicate dubbo header len and compare magic number
func tarsMatcher(data []byte) api.MatchResult {

	if len(data) < MessageSizeLen+IVersionLen {
		return api.MatchAgain
	}
	//check iVersion first.Both requestPackage and responsePackage has iVersion field
	//protocol defines: https://tarscloud.github.io/TarsDocs/base/tars-protocol.html#main-chapter-2
	//iVersion:
	//		const short TARSVERSION  = 0x01;
	//    	const short TUPVERSION  = 0x03;
	// 4 bit tag (1) + 4 bit type (0) + 8 bit data
	if data[IVersionHeaderIdx] == 16 && (data[iVersionDataIdx] == 1 || data[iVersionDataIdx] == 3) {
		pkgLen, status := tarsprotocol.TarsRequest(data)
		if pkgLen == 0 && status == tarsprotocol.PACKAGE_LESS {
			return api.MatchAgain
		}
		if pkgLen == 0 && status == tarsprotocol.PACKAGE_ERROR {
			return api.MatchFailed
		}
		if status == tarsprotocol.PACKAGE_FULL {
			return api.MatchSuccess
		}
	}
	return api.MatchFailed
}
