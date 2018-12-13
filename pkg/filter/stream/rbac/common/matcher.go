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
	"regexp"
	"strings"
)

// HeaderMatcher
type HeaderMatcher interface {
	//	*HeaderMatcher_ExactMatch (supported)
	//	*HeaderMatcher_RegexMatch (supported)
	//	*HeaderMatcher_RangeMatch
	//	*HeaderMatcher_PresentMatch (supported)
	//	*HeaderMatcher_PrefixMatch (supported)
	//	*HeaderMatcher_SuffixMatch (supported)
	HeaderMatcher()
	Equal(string) (bool, error)
}

func (matcher *HeaderMatcherExactMatch) HeaderMatcher()   {}
func (matcher *HeaderMatcherPrefixMatch) HeaderMatcher()  {}
func (matcher *HeaderMatcherSuffixMatch) HeaderMatcher()  {}
func (matcher *HeaderMatcherRegexMatch) HeaderMatcher()   {}
func (matcher *HeaderMatcherPresentMatch) HeaderMatcher() {}

// HeaderMatcher_ExactMatch
type HeaderMatcherExactMatch struct {
	ExactMatch string
}

func (matcher *HeaderMatcherExactMatch) Equal(targetValue string) (bool, error) {
	return matcher.ExactMatch == targetValue, nil
}

// HeaderMatcher_PrefixMatch
type HeaderMatcherPrefixMatch struct {
	PrefixMatch string
}

func (matcher *HeaderMatcherPrefixMatch) Equal(targetValue string) (bool, error) {
	return strings.HasPrefix(targetValue, matcher.PrefixMatch), nil
}

// HeaderMatcher_SuffixMatch
type HeaderMatcherSuffixMatch struct {
	SuffixMatch string
}

func (matcher *HeaderMatcherSuffixMatch) Equal(targetValue string) (bool, error) {
	return strings.HasSuffix(targetValue, matcher.SuffixMatch), nil
}

// HeaderMatcher_RegexMatch
type HeaderMatcherRegexMatch struct {
	RegexMatch *regexp.Regexp
}

func (matcher *HeaderMatcherRegexMatch) Equal(targetValue string) (bool, error) {
	return matcher.RegexMatch.MatchString(targetValue), nil
}

// HeaderMatcher_PresentMatch
type HeaderMatcherPresentMatch struct {
	PresentMatch bool
}

func (matcher *HeaderMatcherPresentMatch) Equal(targetValue string) (bool, error) {
	return matcher.PresentMatch, nil
}
