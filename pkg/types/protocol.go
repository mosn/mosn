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
	"context"
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
	// ProtocolName must be unique
	ProtocolName string
	// MatchResult
	MatchResult int
	// ProtocolMatch recognize if the given data matches the protocol specification or not
	ProtocolMatch func(data []byte) MatchResult
)

const (
	MatchFailed  MatchResult = iota
	MatchSuccess
	MatchAgain
)

// Protocol need to provides ability to convert mode-to-binary and vice-versa
type Protocol interface {
	// Encoder is the encoder implementation of the protocol
	Encoder
	// Decoder is the decoder implementation of the protocol
	Decoder
	// Name is the  name of the protocol
	Name() ProtocolName
}

// HeaderMap is a interface to provide operation facade with user-value headers
type HeaderMap interface {
	// Get value of key
	// If multiple values associated with this key, first one will be returned.
	Get(key string) (string, bool)

	// Set key-value pair in header map, the previous pair will be replaced if exists
	Set(key, value string)

	// Add value for given key.
	// Multiple headers with the same key may be added with this function.
	// Use Set for setting a single header for the given key.
	Add(key, value string)

	// Del delete pair of specified key
	Del(key string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(key, value string) bool)

	// Clone used to deep copy header's map
	Clone() HeaderMap

	// ByteSize return size of HeaderMap
	ByteSize() uint64
}

// ProtocolEngine is a protocols' facade used by Stream, it provides
// auto protocol detection by the ProtocolMatch func
type ProtocolEngine interface {
	// Match use registered matchFunc to recognize corresponding protocol
	Match(ctx context.Context, data IoBuffer) (ProtocolName, MatchResult)
	// Register register encoder and decoder, which recognized by the matchFunc
	Register(matchFunc ProtocolMatch, protocol ProtocolName) error
}

// Encoder is a encoder interface to extend various of protocols
type Encoder interface {
	// Encode encodes a model to binary data
	// return 1. encoded bytes 2. encode error
	Encode(ctx context.Context, model interface{}) (IoBuffer, error)

	// EncodeTo encodes a model to binary data, and append into the given buffer
	// This method should be used in term of performance
	// return 1. encoded bytes number 2. encode error
	//EncodeTo(ctx context.Context, model interface{}, buf IoBuffer) (int, error)
}

// Decoder is a decoder interface to extend various of protocols
type Decoder interface {
	// Decode decodes binary data to a model
	// pass sub protocol type to identify protocol format
	// return 1. decoded model(nil if no enough data) 2. decode error
	Decode(ctx context.Context, data IoBuffer) (interface{}, error)
}
