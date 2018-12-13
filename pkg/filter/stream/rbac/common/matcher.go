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
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

// StringMatcher
type StringMatcher interface {
	//	*StringMatcher_Exact
	//	*StringMatcher_Prefix
	//	*StringMatcher_Suffix
	//	*StringMatcher_Regex
	Equal(string) bool
}

// StringMatcher_Exact
type ExactStringMatcher struct {
	ExactMatch string
}

func (matcher *ExactStringMatcher) Equal(targetValue string) bool {
	return matcher.ExactMatch == targetValue
}

// StringMatcher_Prefix
type PrefixStringMatcher struct {
	PrefixMatch string
}

func (matcher *PrefixStringMatcher) Equal(targetValue string) bool {
	return strings.HasPrefix(targetValue, matcher.PrefixMatch)
}

// StringMatcher_Suffix
type SuffixStringMatcher struct {
	SuffixMatch string
}

func (matcher *SuffixStringMatcher) Equal(targetValue string) bool {
	return strings.HasSuffix(targetValue, matcher.SuffixMatch)
}

// StringMatcher_Regex
type RegexStringMatcher struct {
	RegexMatch *regexp.Regexp
}

func (matcher *RegexStringMatcher) Equal(targetValue string) bool {
	return matcher.RegexMatch.MatchString(targetValue)
}

// HeaderMatcher
type HeaderMatcher interface {
	//	*HeaderMatcher_ExactMatch (supported)
	//	*HeaderMatcher_RegexMatch (supported)
	//	*HeaderMatcher_RangeMatch (supported)
	//	*HeaderMatcher_PresentMatch (supported)
	//	*HeaderMatcher_PrefixMatch (supported)
	//	*HeaderMatcher_SuffixMatch (supported)
	isHeaderMatcher()
	Equal(string) bool
}

func (matcher *ExactStringMatcher) isHeaderMatcher()        {}
func (matcher *PrefixStringMatcher) isHeaderMatcher()       {}
func (matcher *SuffixStringMatcher) isHeaderMatcher()       {}
func (matcher *RegexStringMatcher) isHeaderMatcher()        {}
func (matcher *HeaderMatcherPresentMatch) isHeaderMatcher() {}
func (matcher *HeaderMatcherRangeMatch) isHeaderMatcher()   {}

// HeaderMatcher_PresentMatch
type HeaderMatcherPresentMatch struct {
	PresentMatch bool
}

func (matcher *HeaderMatcherPresentMatch) Equal(targetValue string) bool {
	return matcher.PresentMatch
}

// HeaderMatcher_RangeMatch
type HeaderMatcherRangeMatch struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func (matcher *HeaderMatcherRangeMatch) Equal(targetValue string) bool {
	if intValue, err := strconv.ParseInt(targetValue, 10, 64); err != nil {
		return false
	} else {
		return intValue >= matcher.Start && intValue < matcher.End
	}
}

func NewHeaderMatcher(header *route.HeaderMatcher) (HeaderMatcher, error) {
	switch header.HeaderMatchSpecifier.(type) {
	case *route.HeaderMatcher_ExactMatch:
		return &ExactStringMatcher{
			ExactMatch: header.HeaderMatchSpecifier.(*route.HeaderMatcher_ExactMatch).ExactMatch,
		}, nil
	case *route.HeaderMatcher_PrefixMatch:
		return &PrefixStringMatcher{
			PrefixMatch: header.HeaderMatchSpecifier.(*route.HeaderMatcher_PrefixMatch).PrefixMatch,
		}, nil
	case *route.HeaderMatcher_SuffixMatch:
		return &SuffixStringMatcher{
			SuffixMatch: header.HeaderMatchSpecifier.(*route.HeaderMatcher_SuffixMatch).SuffixMatch,
		}, nil
	case *route.HeaderMatcher_RegexMatch:
		if rePattern, err := regexp.Compile(
			header.HeaderMatchSpecifier.(*route.HeaderMatcher_RegexMatch).RegexMatch); err != nil {
			return nil, fmt.Errorf("[NewHeaderMatcher] failed to build regex, error: %v", err)
		} else {
			return &RegexStringMatcher{
				RegexMatch: rePattern,
			}, nil
		}
	case *route.HeaderMatcher_PresentMatch:
		return &HeaderMatcherPresentMatch{
			PresentMatch: header.HeaderMatchSpecifier.(*route.HeaderMatcher_PresentMatch).PresentMatch,
		}, nil
	case *route.HeaderMatcher_RangeMatch:
		return &HeaderMatcherRangeMatch{
			Start: header.HeaderMatchSpecifier.(*route.HeaderMatcher_RangeMatch).RangeMatch.Start,
			End:   header.HeaderMatchSpecifier.(*route.HeaderMatcher_RangeMatch).RangeMatch.End,
		}, nil
	default:
		return nil, fmt.Errorf(
			"[NewHeaderMatcher] not support HeaderMatchSpecifier type found, detail: %v",
			reflect.TypeOf(header.HeaderMatchSpecifier))
	}
}
