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

package types

import (
	"mosn.io/api/types"
)

// 	 The bunch of interfaces are structure skeleton to build a extensible protocol engine.
//
//   In mosn, we have 4 layers to build a mesh, protocol is the core layer to do protocol related encode/decode.
//	 -----------------------
//   |        PROXY          |
//    -----------------------
//   |       STREAMING       |
//    -----------------------
//   |        PROTOCOL       |
//    -----------------------
//   |         NET/IO        |
//    -----------------------
//
//	 Stream layer leverages protocol's ability to do binary-model conversation. In detail, Stream uses Protocols's encode/decode facade method and DecodeFilter to receive decode event call.
//

type (
	// MatchResult
	// Deprecated: use mosn.io/api/types/protocol.go:MatchResult instead
	MatchResult = types.MatchResult

	// ProtocolMatch recognize if the given data matches the protocol specification or not
	// Deprecated: use mosn.io/api/types/protocol.go:ProtocolMatch instead
	ProtocolMatch = types.ProtocolMatch
)

const (
	MatchFailed  = types.MatchFailed
	MatchSuccess = types.MatchSuccess
	MatchAgain   = types.MatchAgain
)

// TODO: Protocol and api.Protocol have the same name, maybe makes some ambiguity.

// Protocol need to provides ability to convert mode-to-binary and vice-versa
// Deprecated: use mosn.io/api/types/protocol.go:Protocol instead
type Protocol = types.Protocol

// ProtocolEngine is a protocols' facade used by Stream, it provides
// auto protocol detection by the ProtocolMatch func
// Deprecated: use mosn.io/api/types/protocol.go:ProtocolEngine instead
type ProtocolEngine = types.ProtocolEngine

// Encoder is a encoder interface to extend various of protocols
// Deprecated: use mosn.io/api/types/protocol.go:Encoder instead
type Encoder = types.Encoder

// Decoder is a decoder interface to extend various of protocols
// Deprecated: use mosn.io/api/types/protocol.go:Decoder instead
type Decoder = types.Decoder
