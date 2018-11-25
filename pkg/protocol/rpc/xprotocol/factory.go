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
	"errors"
	"strconv"

	networkbuffer "github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	subProtocolFactories map[SubProtocol]CodecFactory
	xProtocolCoder       = &Coder{}
	xProtocolEngine      = rpc.NewEngine(xProtocolCoder, xProtocolCoder)
)

func Engine() types.ProtocolEngine {
	return xProtocolEngine
}

// Coder
// types.Encoder
// types.Decoder
type Coder struct {
}

func (coder *Coder) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	xRpcCmd, ok := model.(XRpcCmd)
	if ok {
		return networkbuffer.NewIoBufferBytes(xRpcCmd.data), nil
	}
	err := errors.New("fail to convert to XRpcCmd")
	return nil, err
}

func (coder *Coder) Decode(ctx context.Context, data types.IoBuffer, subProtocolType string) (interface{}, error) {
	codec := CreateSubProtocolCodec(ctx, SubProtocol(subProtocolType))
	if codec == nil {
		err := errors.New("create sub protocol fail")
		return nil, err
	}
	frames := codec.SplitFrame(data.Bytes())
	xRpcCmd := &XRpcCmd{
		ctx:    ctx,
		codec:  codec,
		data:   nil,
		frames: frames,
		offset: 0,
		header: make(map[string]string),
	}
	return xRpcCmd, nil
}

// Register SubProtocol Plugin
func Register(prot SubProtocol, factory CodecFactory) {
	if subProtocolFactories == nil {
		subProtocolFactories = make(map[SubProtocol]CodecFactory)
	}
	subProtocolFactories[prot] = factory
}

// CreateSubProtocolCodec return SubProtocol Codec
func CreateSubProtocolCodec(context context.Context, prot SubProtocol) Multiplexing {

	if spc, ok := subProtocolFactories[prot]; ok {
		log.DefaultLogger.Tracef("create sub protocol codec %v success", prot)
		return spc.CreateSubProtocolCodec(context)
	}
	log.DefaultLogger.Errorf("unknown sub protocol = %v", prot)
	return nil
}

type XRpcCmd struct {
	ctx    context.Context
	codec  Multiplexing
	data   []byte
	frames [][]byte
	offset uint32
	header map[string]string
}

func (xRpcCmd *XRpcCmd) ProtocolCode() byte {
	// xprotocol unsupport protocol code ,so always return 0
	// xprotocol need sub protocol name
	return 0
}
func (xRpcCmd *XRpcCmd) RequestID() uint64 {
	streamId := xRpcCmd.codec.GetStreamID(xRpcCmd.data)
	requestId, err := strconv.ParseUint(streamId, 10, 64)
	if err != nil {
		log.DefaultLogger.Errorf("get request id fail,streamId = %v", streamId)
		return 0
	}
	return requestId
}
func (xRpcCmd *XRpcCmd) SetRequestID(requestID uint64) {
	streamId := strconv.FormatUint(requestID, 10)
	xRpcCmd.codec.SetStreamID(xRpcCmd.data, streamId)
}
func (xRpcCmd *XRpcCmd) Header() map[string]string {
	return xRpcCmd.header
}

func (xRpcCmd *XRpcCmd) Data() []byte {
	return xRpcCmd.data
}

func (xRpcCmd *XRpcCmd) SetHeader(header map[string]string) {
	xRpcCmd.header = header
}

func (xRpcCmd *XRpcCmd) SetData(data []byte) {
	xRpcCmd.data = data
}

//Multiplexing
func (xRpcCmd *XRpcCmd) SplitFrame(data []byte) [][]byte {
	return xRpcCmd.codec.SplitFrame(data)
}
func (xRpcCmd *XRpcCmd) GetStreamID(data []byte) string {
	return xRpcCmd.codec.GetStreamID(data)
}
func (xRpcCmd *XRpcCmd) SetStreamID(data []byte, streamID string) []byte {
	return xRpcCmd.codec.SetStreamID(data, streamID)
}

//Tracing
func (xRpcCmd *XRpcCmd) GetServiceName(data []byte) string {
	tracingCmd, ok := xRpcCmd.codec.(Tracing)
	if ok {
		return tracingCmd.GetServiceName(data)
	}
	return ""
}
func (xRpcCmd *XRpcCmd) GetMethodName(data []byte) string {
	tracingCmd, ok := xRpcCmd.codec.(Tracing)
	if ok {
		return tracingCmd.GetMethodName(data)
	}
	return ""
}

//RequestRouting
func (xRpcCmd *XRpcCmd) GetMetas(data []byte) map[string]string {
	requestRoutingCmd, ok := xRpcCmd.codec.(RequestRouting)
	if ok {
		return requestRoutingCmd.GetMetas(data)
	}
	return nil
}

//ProtocolConvertor
func (xRpcCmd *XRpcCmd) Convert(data []byte) (map[string]string, []byte) {
	protocolConvertorCmd, ok := xRpcCmd.codec.(ProtocolConvertor)
	if ok {
		return protocolConvertorCmd.Convert(data)
	}
	return nil, nil
}
func (xRpcCmd *XRpcCmd) Get(key string) (string, bool) {
	//current unsupport
	return "", false
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (xRpcCmd *XRpcCmd) Set(key, value string) {
	//current unsupport
}

// Del delete pair of specified key
func (xRpcCmd *XRpcCmd) Del(key string) {
	//current unsupport
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (xRpcCmd *XRpcCmd) Range(f func(key, value string) bool) {
	//current unsupport
}

// ByteSize return size of HeaderMap
func (xRpcCmd *XRpcCmd) ByteSize() uint64 {
	//current unsupport
	return 0
}
