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
	"regexp"
	"strings"
)

// HeaderMatcher
type HeaderMatcher interface {
	//	*HeaderMatcher_ExactMatch (supported)
	//	*HeaderMatcher_RegexMatch (supported)
	//	*HeaderMatcher_RangeMatch
	//	*HeaderMatcher_PresentMatch
	//	*HeaderMatcher_PrefixMatch (supported)
	//	*HeaderMatcher_SuffixMatch (supported)
	HeaderMatcher()
	Equal(interface{}) (bool, error)
}

func (matcher *HeaderMatcherExactMatch) HeaderMatcher()  {}
func (matcher *HeaderMatcherPrefixMatch) HeaderMatcher() {}
func (matcher *HeaderMatcherSuffixMatch) HeaderMatcher() {}
func (matcher *HeaderMatcherRegexMatch) HeaderMatcher()  {}

// HeaderMatcher_ExactMatch
type HeaderMatcherExactMatch struct {
	ExactMatch string
}

func (matcher *HeaderMatcherExactMatch) Equal(targetValue interface{}) (bool, error) {
	if target, ok := targetValue.(string); ok {
		return matcher.ExactMatch == target, nil
	} else {
		return false, fmt.Errorf(
			"[HeaderMatcherExactMatch.Equal] targetValue must be string, received %v", targetValue)
	}
}

// HeaderMatcher_PrefixMatch
type HeaderMatcherPrefixMatch struct {
	PrefixMatch string
}

func (matcher *HeaderMatcherPrefixMatch) Equal(targetValue interface{}) (bool, error) {
	if target, ok := targetValue.(string); ok {
		return strings.HasPrefix(target, matcher.PrefixMatch), nil
	} else {
		return false, fmt.Errorf(
			"[HeaderMatcherPrefixMatch.Equal] targetValue must be string, received %v", targetValue)
	}
}

// HeaderMatcher_SuffixMatch
type HeaderMatcherSuffixMatch struct {
	SuffixMatch string
}

func (matcher *HeaderMatcherSuffixMatch) Equal(targetValue interface{}) (bool, error) {
	if target, ok := targetValue.(string); ok {
		return strings.HasSuffix(target, matcher.SuffixMatch), nil
	} else {
		return false, fmt.Errorf(
			"[HeaderMatcherSuffixMatch.Equal] targetValue must be string, received %v", targetValue)
	}
}

// HeaderMatcher_RegexMatch
type HeaderMatcherRegexMatch struct {
	RegexMatch *regexp.Regexp
}

func (matcher *HeaderMatcherRegexMatch) Equal(targetValue interface{}) (bool, error) {
	if target, ok := targetValue.(string); ok {
		return matcher.RegexMatch.MatchString(target), nil
	} else {
		return false, fmt.Errorf(
			"[HeaderMatcherRegexMatch.Equal] targetValue must be string, received %v", targetValue)
	}

}
