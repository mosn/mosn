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

package tars

import (
	"context"
	"fmt"

	"github.com/TarsCloud/TarsGo/tars"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterProtocol(ProtocolName, &tarsProtocol{})
}

var MagicTag = []byte{0xda, 0xbb}

type tarsProtocol struct{}

func (proto *tarsProtocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto *tarsProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case *Request:
		return encodeRequest(ctx, cmd)
	case *Response:
		return encodeResponse(ctx, cmd)
	default:
		log.Proxy.Errorf(ctx, "[protocol][tars] encode with unknown command : %+v", model)
	}
	return nil, xprotocol.ErrUnknownType
}

func (proto *tarsProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	_, status := tars.TarsRequest(data.Bytes())
	if status == tars.PACKAGE_FULL {
		streamType, err := getStreamType(data.Bytes())
		switch streamType {
		case CmdTypeRequest:
			return decodeRequest(ctx, data)
		case CmdTypeResponse:
			return decodeResponse(ctx, data)
		default:
			// unknown cmd type
			return nil, fmt.Errorf("[protocol][tars] Decode Error, type = %s , err = %v", UnKnownCmdType, err)
		}
	}
	return nil, nil
}

// heartbeater
func (proto *tarsProtocol) Trigger(requestId uint64) xprotocol.XFrame {
	// not support
	return nil
}

func (proto *tarsProtocol) Reply(request xprotocol.XFrame) xprotocol.XRespFrame {
	// not support
	return nil
}

// hijacker
func (proto *tarsProtocol) Hijack(request xprotocol.XFrame, statusCode uint32) xprotocol.XRespFrame {
	// not support
	return nil
}

func (proto *tarsProtocol) Mapping(httpStatusCode uint32) uint32 {
	// not support
	return 0
}

//判断packet的类型，resonse Packet的包tag=5是字段iRet,int类型；request packet的包tag=5是字段sServantName,string类型
func getStreamType(pkg []byte) (byte, error) {
	// skip pkg length
	pkg = pkg[4:]
	b := codec.NewReader(pkg)
	err, have, ty := b.SkipToNoCheck(5, true)
	if err != nil {
		return CmdTypeUndefine, err
	}
	if !have {
		return CmdTypeUndefine, nil
	}
	switch ty {
	case codec.INT, codec.ZERO_TAG:
		return CmdTypeResponse, nil
	case codec.STRING1:
		return CmdTypeRequest, nil
	case codec.STRING4:
		return CmdTypeRequest, nil
	default:
		return CmdTypeUndefine, nil

	}
}
