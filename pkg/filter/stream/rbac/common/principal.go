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
	"regexp"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2alpha"
)

type InheritPrincipal interface {
	InheritPrincipal()
	// A policy matches if and only if at least one of InheritPermission.Match return true
	// AND at least one of InheritPrincipal.Match return true
	Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool
}

func (*PrincipalAny) InheritPrincipal()      {}
func (*PrincipalSourceIp) InheritPrincipal() {}
func (*PrincipalHeader) InheritPrincipal()   {}

// Principal_Any
type PrincipalAny struct {
	Any bool
}

func NewPrincipalAny(principal *v2alpha.Principal_Any) (*PrincipalAny, error) {
	return &PrincipalAny{
		Any: principal.Any,
	}, nil
}

func (principal *PrincipalAny) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	return principal.Any
}

// Principal_SourceIp
type PrincipalSourceIp struct {
	CidrRange *net.IPNet
}

func NewPrincipalSourceIp(principal *v2alpha.Principal_SourceIp) (*PrincipalSourceIp, error) {
	addressPrefix := principal.SourceIp.AddressPrefix
	prefixLen := principal.SourceIp.PrefixLen.GetValue()
	if _, ipNet, err := net.ParseCIDR(addressPrefix + "/" + strconv.Itoa(int(prefixLen))); err != nil {
		return nil, err
	} else {
		inheritPrincipal := &PrincipalSourceIp{
			CidrRange: ipNet,
		}
		return inheritPrincipal, nil
	}
}

func (principal *PrincipalSourceIp) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	remoteAddr := cb.Connection().RemoteAddr().String()
	remoteIP, _, err := parseAddr(remoteAddr)
	if err != nil {
		log.DefaultLogger.Errorf(
			"[PrincipalSourceIp.Match] failed to parse remote address in rbac filter, err: ", err)
		return false
	}
	if principal.CidrRange.Contains(remoteIP) {
		return true
	} else {
		return false
	}
}

// Principal_Header
type PrincipalHeader struct {
	Target      string
	Matcher     HeaderMatcher
	InvertMatch bool
}

func NewPrincipalHeader(principal *v2alpha.Principal_Header) (*PrincipalHeader, error) {
	inheritPrincipal := &PrincipalHeader{}
	inheritPrincipal.Target = principal.Header.Name
	inheritPrincipal.InvertMatch = principal.Header.InvertMatch
	switch principal.Header.HeaderMatchSpecifier.(type) {
	case *route.HeaderMatcher_ExactMatch:
		inheritPrincipal.Matcher = &HeaderMatcherExactMatch{
			ExactMatch: principal.Header.HeaderMatchSpecifier.(*route.HeaderMatcher_ExactMatch).ExactMatch,
		}
	case *route.HeaderMatcher_PrefixMatch:
		inheritPrincipal.Matcher = &HeaderMatcherPrefixMatch{
			PrefixMatch: principal.Header.HeaderMatchSpecifier.(*route.HeaderMatcher_PrefixMatch).PrefixMatch,
		}
	case *route.HeaderMatcher_SuffixMatch:
		inheritPrincipal.Matcher = &HeaderMatcherSuffixMatch{
			SuffixMatch: principal.Header.HeaderMatchSpecifier.(*route.HeaderMatcher_SuffixMatch).SuffixMatch,
		}
	case *route.HeaderMatcher_RegexMatch:
		if rePattern, err := regexp.Compile(
			principal.Header.HeaderMatchSpecifier.(*route.HeaderMatcher_RegexMatch).RegexMatch); err != nil {
			return nil, fmt.Errorf("[NewPrincipalHeader] failed not build regex, error: %v", err)
		} else {
			inheritPrincipal.Matcher = &HeaderMatcherRegexMatch{
				RegexMatch: rePattern,
			}
		}
	case *route.HeaderMatcher_PresentMatch:
		inheritPrincipal.Matcher = &HeaderMatcherPresentMatch{
			PresentMatch: principal.Header.HeaderMatchSpecifier.(*route.HeaderMatcher_PresentMatch).PresentMatch,
		}
	default:
		return nil, fmt.Errorf(
			"[NewPrincipalHeader] not support Principal_Header.Header.HeaderMatchSpecifier type found, detail: %v",
			reflect.TypeOf(principal.Header.HeaderMatchSpecifier))
	}
	return inheritPrincipal, nil
}

func (principal *PrincipalHeader) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	targetValue, found := headerMapper(principal.Target, headers)

	// HeaderMatcherPresentMatch is a little special
	if matcher, ok := principal.Matcher.(*HeaderMatcherPresentMatch); ok {
		// HeaderMatcherPresentMatch matches if and only if header found and PresentMatch is true
		isMatch := found && matcher.PresentMatch
		return principal.InvertMatch != isMatch
	}

	// return false when targetValue is not found, except matcher is `HeaderMatcherPresentMatch`
	if !found {
		return false
	}

	isMatch, err := principal.Matcher.Equal(targetValue)
	if err != nil {
		log.DefaultLogger.Errorf(
			"[NewPrincipalHeader] failed to perform `PrincipalHeader.Matcher.Equal`, error: %v", err)
		return false
	}

	// principal.InvertMatch xor isMatch
	return principal.InvertMatch != isMatch
}

// Receive the v2alpha.Principal input and convert it to mosn rbac principal
func NewInheritPrincipal(principal *v2alpha.Principal) (InheritPrincipal, error) {
	// Types that are valid to be assigned to Identifier:
	//	*Principal_AndIds
	//	*Principal_OrIds
	//	*Principal_Any (supported)
	//	*Principal_Authenticated_
	//	*Principal_SourceIp (supported)
	//	*Principal_Header
	//	*Principal_Metadata
	switch principal.Identifier.(type) {
	case *v2alpha.Principal_Any:
		return NewPrincipalAny(principal.Identifier.(*v2alpha.Principal_Any))
	case *v2alpha.Principal_SourceIp:
		return NewPrincipalSourceIp(principal.Identifier.(*v2alpha.Principal_SourceIp))
	case *v2alpha.Principal_Header:
		return NewPrincipalHeader(principal.Identifier.(*v2alpha.Principal_Header))
	default:
		return nil, fmt.Errorf("[NewInheritPrincipal] not supported Principal.Identifier type found, detail: %v",
			reflect.TypeOf(principal.Identifier))
	}
}
