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
	"mosn.io/pkg/protocol/xprotocol"
)

// RegisterProtocol register the protocol to factory
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:RegisterProtocol instead
var RegisterProtocol = xprotocol.RegisterProtocol

// GetProtocol return the corresponding protocol for given name(if was registered)
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:GetProtocol instead
var GetProtocol = xprotocol.GetProtocol

// RegisterMatcher register the matcher of the protocol into factory
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:RegisterMatcher instead
var RegisterMatcher = xprotocol.RegisterMatcher

// GetMatcher return the corresponding matcher for given name(if was registered)
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:GetMatcher instead
var GetMatcher = xprotocol.GetMatcher

// RegisterMapping register the HTTP status code mapping function of the protocol into factory
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:RegisterMapping instead
var RegisterMapping = xprotocol.RegisterMapping

// GetMapping return the corresponding HTTP status code mapping function for given name(if was registered)
// Deprecated: use mosn.io/pkg/protocol/xprotocol/factory.go:GetMapping instead
var GetMapping = xprotocol.GetMapping
