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

import "strings"

// HeaderMatcher
type HeaderMatcher interface {
	//	*HeaderMatcher_ExactMatch
	//	*HeaderMatcher_RegexMatch
	//	*HeaderMatcher_RangeMatch
	//	*HeaderMatcher_PresentMatch
	//	*HeaderMatcher_PrefixMatch
	//	*HeaderMatcher_SuffixMatch
	HeaderMatcher()
	Equal(interface{}) bool
}

func (matcher *HeaderMatcherExactMatch) HeaderMatcher()  {}
func (matcher *HeaderMatcherPrefixMatch) HeaderMatcher() {}
func (matcher *HeaderMatcherSuffixMatch) HeaderMatcher() {}

// HeaderMatcher_ExactMatch
type HeaderMatcherExactMatch struct {
	ExactMatch string
}

func (matcher *HeaderMatcherExactMatch) Equal(targetValue interface{}) bool {
	return matcher.ExactMatch == targetValue.(string)
}

// HeaderMatcher_PrefixMatch
type HeaderMatcherPrefixMatch struct {
	PrefixMatch string
}

func (matcher *HeaderMatcherPrefixMatch) Equal(targetValue interface{}) bool {
	return strings.HasPrefix(targetValue.(string), matcher.PrefixMatch)
}

// HeaderMatcher_SuffixMatch
type HeaderMatcherSuffixMatch struct {
	SuffixMatch string
}

func (matcher *HeaderMatcherSuffixMatch) Equal(targetValue interface{}) bool {
	return strings.HasSuffix(targetValue.(string), matcher.SuffixMatch)
}
