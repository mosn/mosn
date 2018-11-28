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

package mhttp2

import (
	"github.com/alipay/sofa-mosn/pkg/module/http2"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func EngineServer(sc *http2.MServerConn) types.ProtocolEngine {
	return rpc.NewEngine(&serverCodec{sc: sc}, &serverCodec{sc: sc}, nil)
}

func EngineClient(cc *http2.MClientConn) types.ProtocolEngine {
	return rpc.NewEngine(&clientCodec{cc: cc}, &clientCodec{cc: cc}, nil)
}
