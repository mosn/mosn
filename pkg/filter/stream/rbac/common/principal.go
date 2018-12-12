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

package common

import (
	"fmt"
	"net"
	"reflect"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2alpha"
)

type InheritPrincipal interface {
	InheritPrincipal()
	Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool
}

func (*PrincipalAny) InheritPrincipal()      {}
func (*PrincipalSourceIp) InheritPrincipal() {}

// Principal_Any
type PrincipalAny struct {
	Any bool
}

func (principal *PrincipalAny) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	return principal.Any
}

// Principal_SourceIp
type PrincipalSourceIp struct {
	CidrRange *net.IPNet
}

func (principal *PrincipalSourceIp) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	remoteAddr := cb.Connection().RemoteAddr().String()
	remoteIP, _, err := parseAddr(remoteAddr)
	if err != nil {
		log.DefaultLogger.Errorf("failed to parse remote address in rbac filter, err: ", err)
		return false
	}
	if principal.CidrRange.Contains(remoteIP) {
		return true
	} else {
		return false
	}
}

// Receive the v2alpha.Principal input and convert it to mosn rbac principal
func NewInheritPrincipal(principal *v2alpha.Principal) (InheritPrincipal, error) {
	// Types that are valid to be assigned to Identifier:
	//	*Principal_AndIds
	//	*Principal_OrIds
	//	*Principal_Any
	//	*Principal_Authenticated_
	//	*Principal_SourceIp
	//	*Principal_Header
	//	*Principal_Metadata
	switch principal.Identifier.(type) {
	case *v2alpha.Principal_Any:
		inheritPrincipal := new(PrincipalAny)
		inheritPrincipal.Any = principal.Identifier.(*v2alpha.Principal_Any).Any
		return inheritPrincipal, nil
	case *v2alpha.Principal_SourceIp:
		inheritPrincipal := new(PrincipalSourceIp)
		addressPrefix := principal.Identifier.(*v2alpha.Principal_SourceIp).SourceIp.AddressPrefix
		prefixLen := principal.Identifier.(*v2alpha.Principal_SourceIp).SourceIp.PrefixLen.GetValue()
		if _, ipNet, err := net.ParseCIDR(addressPrefix + "/" + strconv.Itoa(int(prefixLen))); err != nil {
			return nil, err
		} else {
			inheritPrincipal.CidrRange = ipNet
			return inheritPrincipal, nil
		}
	default:
		return nil, fmt.Errorf("not supported principal type found, detail: %v", reflect.TypeOf(principal.Identifier))
	}
}
