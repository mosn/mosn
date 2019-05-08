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

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
)

var (
	subProtocolFactories map[SubProtocol]CodecFactory
	xProtocolCoder       = &Coder{}
	xProtocolEngine      = rpc.NewEngine(xProtocolCoder, xProtocolCoder, nil)
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
	xRpcCmd, ok := model.(*XRpcCmd)
	if ok {
		return xRpcCmd.data, nil
	}
	err := errors.New("fail to convert to XRpcCmd")
	return nil, err
}

func (coder *Coder) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	subProtocolType := SubProtocol(mosnctx.Get(ctx, types.ContextSubProtocol).(string))
	codec := CreateSubProtocolCodec(ctx, SubProtocol(subProtocolType))
	if codec == nil {
		err := errors.New("create sub protocol fail")
		return nil, err
	}
	xRpcCmd := &XRpcCmd{
		ctx:    ctx,
		codec:  codec,
		data:   nil,
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

// CreateSubProtocolCodec return SubProtocol Client
func CreateSubProtocolCodec(context context.Context, prot SubProtocol) Multiplexing {

	if spc, ok := subProtocolFactories[prot]; ok {
		log.DefaultLogger.Tracef("create sub protocol codec %v success", prot)
		return spc.CreateSubProtocolCodec(context)
	}
	log.DefaultLogger.Errorf("unknown sub protocol = %v", prot)
	return nil
}

// TODO should change the multiplexing interface to support decode into a data and header
// XRpcCmd In XProtocol Mode , XRpcCmd is a codec wrapper , so data and header is useless currently
type XRpcCmd struct {
	ctx    context.Context
	codec  Multiplexing
	data   types.IoBuffer
	header map[string]string
}

// ProtocolEngine no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) ProtocolCode() byte {
	// xprotocol unsupport protocol code ,so always return 0
	// xprotocol need sub protocol name
	return 0
}

// RequestID no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) RequestID() uint64 {
	streamId := xRpcCmd.codec.GetStreamID(xRpcCmd.data.Bytes())
	requestId, err := strconv.ParseUint(streamId, 10, 64)
	if err != nil {
		log.DefaultLogger.Errorf("get request id fail,streamId = %v", streamId)
		return 0
	}
	return requestId
}

// SetRequestID no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) SetRequestID(requestID uint64) {
	streamId := strconv.FormatUint(requestID, 10)
	xRpcCmd.data = networkbuffer.NewIoBufferBytes(xRpcCmd.codec.SetStreamID(xRpcCmd.data.Bytes(), streamId))
}

// Header no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) Header() map[string]string {
	return xRpcCmd.header
}

// Data no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) Data() types.IoBuffer {
	return xRpcCmd.data
}

// SetHeader no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) SetHeader(header map[string]string) {
	xRpcCmd.header = header
}

// SetData no use util we change multiplexing interface
func (xRpcCmd *XRpcCmd) SetData(data types.IoBuffer) {
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

func (xRpcCmd *XRpcCmd) Get(key string) (value string, ok bool) {
	value, ok = xRpcCmd.header[key]
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (xRpcCmd *XRpcCmd) Set(key string, value string) {
	xRpcCmd.header[key] = value
}

// Del delete pair of specified key
func (xRpcCmd *XRpcCmd) Del(key string) {
	delete(xRpcCmd.header, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (xRpcCmd *XRpcCmd) Range(f func(key, value string) bool) {
	for k, v := range xRpcCmd.header {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

// Clone used to deep copy header's map
func (xRpcCmd *XRpcCmd) Clone() types.HeaderMap {
	return nil
}

// ByteSize return size of HeaderMap
func (xRpcCmd *XRpcCmd) ByteSize() (size uint64) {
	for k, v := range xRpcCmd.header {
		size += uint64(len(k) + len(v))
	}
	return size
}
