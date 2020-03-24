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

package example

import (
	"mosn.io/mosn/pkg/types"
)

// protocol constants
const (
	ProtocolName types.ProtocolName = "x_example" // protocol

	Magic       byte = 'x' //magic
	DirRequest  byte = 0   // dir
	DirResponse byte = 1   // dir

	TypeHeartbeat byte = 0 // cmd code
	TypeMessage   byte = 1
	TypeGoAway    byte = 2

	ResponseStatusSuccess uint16 = 0 // 0x00 response status
	ResponseStatusError   uint16 = 1 // 0x01

	RequestHeaderLen  int = 11 // protocol header fields length
	ResponseHeaderLen int = 13
	MinimalDecodeLen  int = RequestHeaderLen // minimal length for decoding

	RequestIdIndex = 3
)
