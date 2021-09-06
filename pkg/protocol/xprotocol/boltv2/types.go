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

package boltv2

import (
	"mosn.io/mosn/pkg/types"
)

// boltv2 constants
const (
	ProtocolName    types.ProtocolName = "boltv2" // protocol
	ProtocolCode    byte               = 2
	ProtocolVersion byte               = 1 // see details in: https://github.com/sofastack/sofa-bolt/blob/master/src/main/java/com/alipay/remoting/rpc/protocol/RpcCommandEncoderV2.java

	RequestHeaderLen  int = 24 // protocol header fields length
	ResponseHeaderLen int = 22
	LessLen           int = ResponseHeaderLen // minimal length for decoding

	RequestIdIndex         = 6
	RequestHeaderLenIndex  = 18
	ResponseHeaderLenIndex = 16

	ProtocolVersion1 byte = 0x01 // define in https://github.com/sofastack/sofa-bolt/blob/master/src/main/java/com/alipay/remoting/rpc/protocol/RpcProtocolV2.java
)
