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

package xprotocol

import (
	"container/list"
	"errors"
	"math"

	"mosn.io/mosn/pkg/protocol"

	"mosn.io/mosn/pkg/types"
)

type SortedProtocolMatch struct {
	ProtocolName types.ProtocolName
	Matcher      types.ProtocolMatch
	order        int
}

var (
	protocolMap       = make(map[types.ProtocolName]XProtocol)
	matcherMap        = make(map[types.ProtocolName]types.ProtocolMatch)
	sortedMatcherList = list.New()
	mappingMap        = make(map[types.ProtocolName]protocol.HTTPMapping)
)

// RegisterProtocol register the protocol to factory
func RegisterProtocol(name types.ProtocolName, protocol XProtocol) error {
	// check name conflict
	_, ok := protocolMap[name]
	if ok {
		return errors.New("duplicate protocol register:" + string(name))
	}

	protocolMap[name] = protocol
	return nil
}

// GetProtocol return the corresponding protocol for given name(if was registered)
func GetProtocol(name types.ProtocolName) XProtocol {
	return protocolMap[name]
}

// RegisterMatcher register the matcher of the protocol into factory
func RegisterMatcher(name types.ProtocolName, matcher types.ProtocolMatch) error {
	// check name conflict
	_, ok := matcherMap[name]
	if ok {
		return errors.New("duplicate matcher register:" + string(name))
	}

	matcherMap[name] = matcher
	sortedMatcherList.PushBack(SortedProtocolMatch{name, matcher, math.MaxInt64})
	return nil
}

// RegisterMatcher register the matcher of the protocol into factory.
// Order defines the sort order for the matcher. Lower values have higher priority
func RegisterMatcherWithOrder(name types.ProtocolName, matcher types.ProtocolMatch, order int) error {
	// check name conflict
	_, ok := matcherMap[name]
	if ok {
		return errors.New("duplicate matcher register:" + string(name))
	}
	matcherMap[name] = matcher
	//sort list by order
	for e := sortedMatcherList.Front(); e != nil; e = e.Next() {
		if order <= e.Value.(SortedProtocolMatch).order {
			sortedMatcherList.InsertBefore(SortedProtocolMatch{name, matcher, order}, e)
			return nil
		}
	}
	sortedMatcherList.PushBack(SortedProtocolMatch{name, matcher, order})
	return nil
}

// GetMatcher return the corresponding matcher for given name(if was registered)
func GetMatcher(name types.ProtocolName) types.ProtocolMatch {
	return matcherMap[name]
}

//GetMatchers return all matchers that was registered
func GetMatchers() *list.List {
	return sortedMatcherList
}

// RegisterMapping register the HTTP status code mapping function of the protocol into factory
func RegisterMapping(name types.ProtocolName, mapping protocol.HTTPMapping) error {
	// check name conflict
	_, ok := mappingMap[name]
	if ok {
		return errors.New("duplicate mapping register:" + string(name))
	}

	mappingMap[name] = mapping
	return nil
}

// GetMapping return the corresponding HTTP status code mapping function for given name(if was registered)
func GetMapping(name types.ProtocolName) protocol.HTTPMapping {
	return mappingMap[name]
}
