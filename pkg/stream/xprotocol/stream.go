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

	"mosn.io/api"

	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/track"
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

	frame api.XFrame
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
	frame, ok := headers.(api.XFrame)
	if !ok {
		err = fmt.Errorf("headers %T is not a XFrame instance", headers)
		return
	}

	// hijack process
	if s.direction == stream.ServerStream && frame.GetStreamType() == api.Request {
		s.frame, err = s.buildHijackResp(ctx, frame, headers)
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

func (s *xStream) buildHijackResp(ctx context.Context, request api.XFrame, header types.HeaderMap) (api.XFrame, error) {
	status, err := variable.GetString(s.ctx, types.VarHeaderStatus)
	if err != nil {
		return nil, err
	}
	if status != "" {
		statusCode, _ := strconv.Atoi(status)
		proto := s.sc.protocol
		return proto.Hijack(ctx, request, proto.Mapping(uint32(statusCode))), nil
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
		log.Proxy.Debugf(s.ctx, "[stream] [xprotocol] connection %d endStream, direction = %d, requestId = %v", s.sc.netConn.ID(), s.direction, s.id)
	}

	if s.frame != nil {
		// replace requestID
		s.frame.SetRequestId(s.id)

		buf, err := s.sc.protocol.Encode(s.ctx, s.frame)
		if err != nil {
			log.Proxy.Errorf(s.ctx, "[stream] [xprotocol] encode error:%s, requestId = %v", err.Error(), s.id)
			s.ResetStream(types.StreamLocalReset)
			return
		}

		tracks := track.TrackBufferByContext(s.ctx).Tracks

		tracks.StartTrack(track.NetworkDataWrite)
		err = s.sc.netConn.Write(buf)
		tracks.EndTrack(track.NetworkDataWrite)

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
