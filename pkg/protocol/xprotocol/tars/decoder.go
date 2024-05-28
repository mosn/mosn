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

	tarsprotocol "github.com/TarsCloud/TarsGo/tars/protocol"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"github.com/juju/errors"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func decodeRequest(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	frameLen, status := tarsprotocol.TarsRequest(data.Bytes())
	if status != tarsprotocol.PACKAGE_FULL {
		return nil, errors.New("tars request status fail")
	}
	req := &Request{
		CommonHeader: protocol.CommonHeader{},
	}
	rawData := make([]byte, frameLen)
	copy(rawData, data.Bytes()[:frameLen])
	req.rawData = rawData
	req.data = buffer.NewIoBufferBytes(req.rawData)

	// notice: read-only!!! do not modify the raw data!!!
	variable.Set(ctx, types.VarRequestRawData, req.rawData)

	reqPacket := &requestf.RequestPacket{}
	is := codec.NewReader(data.Bytes())
	err = reqPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	req.cmd = reqPacket
	// service aware
	metaHeader, err := getServiceAwareMeta(req)
	for k, v := range metaHeader {
		req.Set(k, v)
	}
	data.Drain(frameLen)
	return req, nil
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	frameLen, status := tarsprotocol.TarsRequest(data.Bytes())
	if status != tarsprotocol.PACKAGE_FULL {
		return nil, errors.New("tars request status fail")
	}
	resp := &Response{}
	rawData := make([]byte, frameLen)
	copy(rawData, data.Bytes()[:frameLen])
	resp.rawData = rawData
	resp.data = buffer.NewIoBufferBytes(resp.rawData)

	// notice: read-only!!! do not modify the raw data!!!
	variable.Set(ctx, types.VarResponseRawData, resp.rawData)

	respPacket := &requestf.ResponsePacket{}
	is := codec.NewReader(data.Bytes())
	err = respPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	resp.cmd = respPacket
	data.Drain(frameLen)
	return resp, nil
}

func getServiceAwareMeta(request *Request) (map[string]string, error) {
	meta := make(map[string]string, 0)
	meta[ServiceNameHeader] = request.cmd.SServantName
	meta[MethodNameHeader] = request.cmd.SFuncName
	return meta, nil
}
