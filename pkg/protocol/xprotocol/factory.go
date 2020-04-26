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
	"errors"

	"mosn.io/mosn/pkg/protocol"

	"mosn.io/mosn/pkg/types"
)

var (
	protocolMap = make(map[types.ProtocolName]XProtocol)
	matcherMap  = make(map[types.ProtocolName]types.ProtocolMatch)
	mappingMap  = make(map[types.ProtocolName]protocol.HTTPMapping)
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
	return nil
}

// GetMatcher return the corresponding matcher for given name(if was registered)
func GetMatcher(name types.ProtocolName) types.ProtocolMatch {
	return matcherMap[name]
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
