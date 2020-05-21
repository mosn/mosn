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

package http2

import (
	"context"

	"mosn.io/mosn/pkg/module/http2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

// types.Encoder
// types.Decoder
type serverCodec struct {
	sc      *http2.MServerConn
	preface bool
	init    bool
}

func (c *serverCodec) Name() types.ProtocolName {
	return protocol.HTTP2
}

func (c *serverCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	ms := model.(*http2.MStream)
	err := ms.SendResponse()
	return nil, err
}

func (c *serverCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if !c.init {
		c.init = true
		c.sc.Init()
	}
	if !c.preface {
		if err := c.sc.Framer.ReadPreface(data); err == nil {
			c.preface = true
		} else {
			return nil, err
		}
	}
	frame, _, err := c.sc.Framer.ReadFrame(ctx, data, 0)
	return frame, err
}

type clientCodec struct {
	cc *http2.MClientConn
}

func (c *clientCodec) Name() types.ProtocolName {
	return protocol.HTTP2
}

func (c *clientCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	ms := model.(*http2.MClientStream)
	err := ms.RoundTrip(ctx)
	return nil, err
}

func (c *clientCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	frame, _, err := c.cc.Framer.ReadFrame(ctx, data, 0)
	return frame, err
}
