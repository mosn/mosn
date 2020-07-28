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
	"context"
	"fmt"
	"strconv"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

// types.Stream
// types.StreamSender
type xStream struct {
	stream.BaseStream

	id        uint64
	direction stream.StreamDirection // 0: out, 1: in
	ctx       context.Context
	sc        *streamConn

	connReset bool

	receiver types.StreamReceiveListener

	frame xprotocol.XFrame
}

// ~~ types.Stream
func (s *xStream) ID() uint64 {
	return s.id
}

// types.StreamSender
func (s *xStream) AppendHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) (err error) {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "[stream] [xprotocol] appendHeaders, direction = %d, requestId = %d", s.direction, s.id)
	}

	// type assertion
	frame, ok := headers.(xprotocol.XFrame)
	if !ok {
		err = fmt.Errorf("headers %T is not a XFrame instance", headers)
		return
	}

	// hijack process
	if s.direction == stream.ServerStream && frame.GetStreamType() == xprotocol.Request {
		s.frame, err = s.buildHijackResp(frame, headers)
		if err != nil {
			return
		}
	} else {
		s.frame = frame
	}

	// endStream
	if endStream {
		s.endStream()
	}
	return
}

func (s *xStream) buildHijackResp(request xprotocol.XFrame, header types.HeaderMap) (xprotocol.XFrame, error) {
	if status, ok := header.Get(types.HeaderStatus); ok {
		header.Del(types.HeaderStatus)
		statusCode, _ := strconv.Atoi(status)
		proto := s.sc.protocol
		return proto.Hijack(request, proto.Mapping(uint32(statusCode))), nil
	}

	return nil, types.ErrNoStatusCodeForHijack
}

func (s *xStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "[stream] [xprotocol] appendData, direction = %d, requestId = %d", s.direction, s.id)
	}

	s.frame.SetData(data)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *xStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	s.endStream()

	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request
func (s *xStream) endStream() {
	defer func() {
		if s.direction == stream.ServerStream {
			s.DestroyStream()
		}
	}()

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "[stream] [xprotocol] endStream, direction = %d, requestId = %v", s.direction, s.id)
	}

	if s.frame != nil {
		// replace requestID
		s.frame.SetRequestId(s.id)

		// remove injected headers
		if _, ok := s.frame.(xprotocol.ServiceAware); ok {
			s.frame.GetHeader().Del(types.HeaderRPCService)
			s.frame.GetHeader().Del(types.HeaderRPCMethod)
		}

		buf, err := s.sc.protocol.Encode(s.ctx, s.frame)
		if err != nil {
			log.Proxy.Errorf(s.ctx, "[stream] [xprotocol] encode error:%s, requestId = %v", err.Error(), s.id)
			s.ResetStream(types.StreamLocalReset)
			return
		}

		err = s.sc.netConn.Write(buf)

		if err != nil {
			log.Proxy.Errorf(s.ctx, "[stream] [xprotocol] endStream, requestId = %v, error = %v", s.id, err)
			if err == types.ErrConnectionHasClosed {
				s.ResetStream(types.StreamConnectionFailed)
			} else {
				s.ResetStream(types.StreamLocalReset)
			}
		}
	}
}

func (s *xStream) GetStream() types.Stream {
	return s
}

func (s *xStream) ResetStream(reason types.StreamResetReason) {
	if s.direction == stream.ClientStream && !s.connReset {
		s.sc.clientMutex.Lock()
		delete(s.sc.clientStreams, s.id)
		s.sc.clientMutex.Unlock()
	}

	s.BaseStream.ResetStream(reason)
}
