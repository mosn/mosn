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

package dubbothrift

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync/atomic"

	"mosn.io/api"

	"mosn.io/mosn/pkg/protocol"

	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/pkg/buffer"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

/**
 * Thrift framed protocol codec for dubbo.
 *
 * |<-                                        message header                                                   ->|<- message body ->|
 * +----------------------+----------------+----------------------+------------------+---------------------------+------------------+
 *  message size (4 byte) | magic (2 bytes)|message size (4 bytes)|head size(2 bytes)| version (1 byte) | header |   message body   |
 * +----------------------+----------------+----------------------+------------------+---------------------------+------------------+
 *                        |<-                                           message size                                              ->|
 *
 * header fields in version 1:
 *   string - service name
 *   long   - dubbo request id
 */
var MagicTag = []byte{0xda, 0xbc}

type thriftProtocol struct{}

func (proto thriftProtocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto thriftProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	if frame, ok := model.(*Frame); ok {
		if frame.Direction == EventRequest {
			return encodeRequest(ctx, frame)
		} else if frame.Direction == EventResponse {
			return encodeResponse(ctx, frame)
		}
	}
	log.Proxy.Errorf(ctx, "[protocol][thrift] encode with unknown command : %+v", model)
	return nil, api.ErrUnknownType
}

func (proto thriftProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if data.Len() >= MessageLenSize+MagicLen {
		// check frame size
		frameLen := binary.BigEndian.Uint32(data.Bytes()[0:MessageLenSize])
		if data.Len() >= int(frameLen) {
			frame, err := decodeFrame(ctx, data)
			if err != nil {
				// unknown cmd type
				return nil, fmt.Errorf("[protocol][thrift] Decode Error, type = %s , err = %v", UnKnownCmdType, err)
			}
			return frame, err
		}
	}
	return nil, nil
}

// heartbeater
func (proto thriftProtocol) Trigger(ctx context.Context, requestId uint64) api.XFrame {
	// not support
	return nil
}

func (proto thriftProtocol) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
	// wherever, heartbeat is not support
	return &Frame{
		Header: Header{
			Magic:         MagicTag,
			Id:            request.GetRequestId(),
			HeaderLength:  0,
			MessageLength: 0,
		},
		payload: []byte{},
	}
}

// https://dubbo.apache.org/zh-cn/blog/dubbo-protocol.html
// hijacker
func (proto thriftProtocol) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {

	frame := request.(*Frame)

	bufferBytes := buffer.NewIoBuffer(1024)
	transport := thrift.NewStreamTransportW(bufferBytes)
	defer transport.Close()
	tProtocol := thrift.NewTBinaryProtocolTransport(transport)

	status := dubboMosnStatusMap[int(statusCode)]
	methodName, _ := frame.Get(MethodNameHeader)
	seqId, _ := frame.Get(SeqIdNameHeader)
	iSeqId, _ := strconv.Atoi(seqId)

	tProtocol.WriteMessageBegin(methodName, thrift.EXCEPTION, int32(iSeqId))
	appException := thrift.NewTApplicationException(status.Status, status.Msg)
	appException.Write(tProtocol)
	tProtocol.WriteMessageEnd()
	tProtocol.Flush(nil)
	transport.Close()
	message := bufferBytes.Bytes()

	log.DefaultLogger.Infof("hijack: %v,  %v", request, statusCode)

	retFrame := &Frame{
		Header: Header{
			FrameLength:   0,
			Magic:         MagicTag,
			MessageLength: 0,
			HeaderLength:  0,
			Version:       1,
			Id:            frame.GetRequestId(),
			Direction:     EventResponse,
			CommonHeader:  protocol.CommonHeader{},
		},
		payload: message,
	}
	frame.Range(func(k, v string) bool {
		retFrame.Set(k, v)
		return true
	})
	return retFrame
}

func (proto thriftProtocol) Mapping(httpStatusCode uint32) uint32 {
	return httpStatusCode
}

// PoolMode returns whether pingpong or multiplex
func (proto thriftProtocol) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (proto thriftProtocol) EnableWorkerPool() bool {
	return true
}

func (proto thriftProtocol) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}
