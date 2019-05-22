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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"sofastack.io/sofa-mosn/pkg/protocol/rpc/xprotocol"
	"log"
	"strconv"
)

func init() {
	xprotocol.Register("tars", &pluginTarsFactory{})
}

type pluginTarsFactory struct {
}

func (ref *pluginTarsFactory) CreateSubProtocolCodec(context context.Context) xprotocol.Multiplexing {
	return NewRpcTars()
}

type rpcTars struct {
}

func NewRpcTars() xprotocol.Tracing {
	return &rpcTars{}
}

const (
	TARS_HEADER_LEN = 4
)

func (r *rpcTars) SplitFrame(data []byte) [][]byte {
	var frames [][]byte
	start := 0
	dataLen := len(data)
	for true {
		frameLen := getTarsLen(data[start:])
		if frameLen > 0 && dataLen >= frameLen {
			// there is one valid tars request
			frames = append(frames, data[start:(start+frameLen)])
			start += frameLen
			dataLen -= frameLen
			if dataLen == 0 {
				// finish
				//fmt.Printf("[SplitFrame] finish\n")
				break
			}
		} else {
			// invalid data
			fmt.Printf("[SplitFrame] over! frameLen=%d, dataLen=%d. frame_cnt=%d\n", frameLen, dataLen, len(frames))
			break
		}
	}
	return frames
}

func (r *rpcTars) GetStreamID(data []byte) string {
	pkg, status := getPacket(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		log.Printf("[ERROR] get stream id failed!cause decode packet error!status=%d, data:%v  \n", status, data)
		return ""
	}
	pType := getPacketType(pkg)
	if pType == UNDEIFINE {
		log.Printf("[ERROR] get stream id failed!cause packet type undifine. data:%v  \n", data)
		return ""
	}
	var tag byte
	if pType == REQUEST {
		tag = 4
	} else {
		tag = 3
	}
	id, err := getRequestId(pkg, tag)
	if err != nil {
		log.Printf("[ERROR] get stream id failed!cause get request id failed! err=%s , data:%v  \n", err.Error(), data)
		return ""
	}
	return strconv.FormatInt(int64(id), 10)
}

func (r *rpcTars) SetStreamID(data []byte, streamID string) []byte {
	pkg, status := getPacket(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		log.Printf("[ERROR] set stream id failed!cause decode packet error!status=%d, data:%v  \n", status, data)
		return data
	}
	pType := getPacketType(pkg)
	if pType == UNDEIFINE {
		log.Printf("[ERROR] set stream id failed!cause packet type undifine. data:%v  \n", data)
		return data
	}
	if pType == REQUEST {
		req := getRequestPacket(pkg)
		if req == nil {
			return data
		}
		reqID, err := strconv.ParseInt(streamID, 10, 64)
		if err != nil {
			log.Printf("[ERROR] set stream id failed!cause parse streamId to int failed!err=%s  id:%s  \n", err.Error(), streamID)
			return data
		}
		req.IRequestId = int32(reqID)
		return req2Byte(req)
	} else {
		resp := getResponsePacket(pkg)
		if resp == nil {
			return data
		}
		respId, err := strconv.ParseInt(streamID, 10, 64)
		if err != nil {
			log.Printf("[ERROR] set stream id failed!cause parse streamId to int failed!err=%s  id:%s  \n", err.Error(), streamID)
			return data
		}
		resp.IRequestId = int32(respId)
		return rsp2Byte(resp)
	}
}

func (r *rpcTars) GetServiceName(data []byte) string {
	pkg, status := getPacket(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		log.Printf("[ERROR] get service name failed!cause decode packet error!status=%d, data:%v  \n", status, data)
		return ""
	}
	pType := getPacketType(pkg)
	if pType == UNDEIFINE {
		log.Printf("[ERROR] get service name failed!cause packet type undifine. data:%v  \n", data)
		return ""
	}
	if pType == RESPONSE {
		//log.Printf("[WARN] get service name return empty!cause packet type is response. data:%v  \n", data)
		return ""
	}
	sn, err := getStringFromPack(pkg, 5)
	if err != nil {
		log.Printf("[ERROR] get service name failed!err=%s, data:%v  \n", err.Error(), data)
		return ""
	}
	return sn
}

func (r *rpcTars) GetMethodName(data []byte) string {
	pkg, status := getPacket(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		log.Printf("[ERROR] get methon name failed!cause decode packet error!status=%d, data:%v  \n", status, data)
		return ""
	}
	pType := getPacketType(pkg)
	if pType == UNDEIFINE {
		log.Printf("[ERROR] get methon name failed!cause packet type undifine. data:%v  \n", data)
		return ""
	}
	if pType == RESPONSE {
		//log.Printf("[WARN] get methon name return empty!cause packet type is response. data:%v  \n", data)
		return ""
	}
	mn, err := getStringFromPack(pkg, 6)
	if err != nil {
		log.Printf("[ERROR] get  methon name failed!err=%s, data:%v  \n", err.Error(), data)
		return ""
	}
	return mn
}

func getRequestPacket(pkg []byte) *requestf.RequestPacket {
	reqPackage := requestf.RequestPacket{}
	is := codec.NewReader(pkg)
	reqPackage.ReadFrom(is)
	return &reqPackage
}

func getResponsePacket(pkg []byte) *requestf.ResponsePacket {
	resonsePacket := requestf.ResponsePacket{}
	is := codec.NewReader(pkg)
	resonsePacket.ReadFrom(is)
	return &resonsePacket
}

func rsp2Byte(rsp *requestf.ResponsePacket) []byte {
	os := codec.NewBuffer()
	rsp.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return sbuf.Bytes()
}

func req2Byte(req *requestf.RequestPacket) []byte {
	os := codec.NewBuffer()
	req.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return sbuf.Bytes()
}

func getTarsLen(data []byte) int {
	pkgLen, status := tars.TarsRequest(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		return -1
	}
	return pkgLen
}

const (
	REQUEST   = 1
	RESPONSE  = 2
	UNDEIFINE = 3
)

func getRequestId(pkg []byte, tag byte) (int32, error) {
	b := codec.NewReader(pkg)
	var requestId int32
	err := b.Read_int32(&requestId, tag, true)
	return requestId, err
}

func getStringFromPack(pkg []byte, tag byte) (string, error) {
	b := codec.NewReader(pkg)
	var data string
	err := b.Read_string(&data, tag, true)
	return data, err
}

//判断packet的类型，resonse Packet的包tag=5是字段iRet,int类型；request packet的包tag=5是字段sServantName,string类型
func getPacketType(pkg []byte) int {
	b := codec.NewReader(pkg)
	err, have, ty := b.SkipToNoCheck(5, true)
	if err != nil {
		log.Printf("[GetPacketType] skip to tag 5 fail!err=%s \n", err.Error())
		return UNDEIFINE
	}
	if !have {
		return UNDEIFINE
	}
	switch ty {
	case codec.ZERO_TAG:
		return RESPONSE
	case codec.BYTE:
		return RESPONSE
	case codec.SHORT:
		return RESPONSE
	case codec.INT:
		return RESPONSE
	case codec.STRING1:
		return REQUEST
	case codec.STRING4:
		return REQUEST
	default:
		return UNDEIFINE

	}
	return UNDEIFINE

}
func getPacket(data []byte) ([]byte, int) {
	pkgLen, status := tars.TarsRequest(data)
	if status == tars.PACKAGE_LESS || status == tars.PACKAGE_ERROR {
		return data, status
	}
	pkg := make([]byte, pkgLen-4)
	copy(pkg, data[4:pkgLen])
	return pkg, status
}
