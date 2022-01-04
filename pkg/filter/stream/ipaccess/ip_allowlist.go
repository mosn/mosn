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
	"net"
)

// IPAllowlist check whether the ip can be accessed
type IPAllowlist struct {
	*IpList
}

// IsAllow checks if provided request is allow by the ipaccess.
func (i *IPAllowlist) IsAllow(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ok, err := i.Exist(host)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	return true, nil
}
func (i *IPAllowlist) IsDenyAction() bool {
	return false
}
