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

package subprotocol

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/stream/xprotocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	xprotocol.Register("X-hsf", &pluginHSFFactory{})
}

type pluginHSFFactory struct{}

func (ref *pluginHSFFactory) CreateSubProtocolCodec(context context.Context) types.Multiplexing {
	return NewRPCHSF()
}

type rpcHSF struct{}

func NewRPCHSF() types.Multiplexing {
	return &rpcHSF{}
}

/**
 * HSF protocol
 * Request
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |magic| ver | type|seria|  externds bytes |        requst id                              |     |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |timeout mills    | service name length   | method name length    | methods args number   | para|
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |meter type length| arguments bytes length| request props length  |                             |
 * +-----------+-----------+-----------------------------------------+
 * |payload: service_name + method_name + parameter_type + argument + request_props                |
 * +-----------------------------------------------------------------------------------------------+
 *
 * parameter_type_length & argument_bytes_lenth are optional
 *
 * Response
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |magic| ver | type|statu|seria|  externds bytes |        requst id                              |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |      body length      |                                                                       |
 * +-----------+-----------+
 * |                               payload                                                         |
 * +-----------------------------------------------------------------------------------------------+
 */

const (
	HSF_REQ_HEADER_LEN = 35
	HSF_RSP_HEADER_LEN = 20
	HSF_MAGIC_IDX      = 0
	HSF_VERSION_IDX    = 1
	HSF_TYPE_IDX       = 2
	HSF_TYPE_REQ       = 0
	HSF_TYPE_RSP       = 1
	HSF_REQ_REQ_ID_IDX = 7
	HSF_RSP_REQ_ID_IDX = 8
	HSF_REQ_ID_LEN     = 8

	HSF_MAGIC_TAG  = 0x0e
	HSF_VERSION_V1 = 1
)

func getHSFReqLen(data []byte) int {
	dataLen := len(data)
	if dataLen < HSF_REQ_HEADER_LEN || data[HSF_MAGIC_IDX] != HSF_MAGIC_TAG || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
		// illegal
		fmt.Printf("[getHSFReqLen] illegal(len=%d): %v\n", dataLen, data)
		return -1
	}
	serviceNameLen := binary.BigEndian.Uint32(data[19:(19 + 4)])
	methodNameLen := binary.BigEndian.Uint32(data[23:(23 + 4)])
	paraTypeNum := binary.BigEndian.Uint32(data[27:(27 + 4)])
	idx := 31
	paraLen := uint32(0)
	for i := uint32(0); i < paraTypeNum; i++ {
		paraTypeNameLen := binary.BigEndian.Uint32(data[idx:(idx + 4)])
		idx += 4
		paraLen += 4 + paraTypeNameLen
	}
	for i := uint32(0); i < paraTypeNum; i++ {
		argLen := binary.BigEndian.Uint32(data[idx:(idx + 4)])
		idx += 4
		paraLen += 4 + argLen
	}
	propsLen := binary.BigEndian.Uint32(data[idx:(idx + 4)])
	reqLen := uint32(HSF_REQ_HEADER_LEN) + serviceNameLen + methodNameLen + paraLen + propsLen
	if uint32(dataLen) < reqLen {
		// illegal
		fmt.Printf("[getHSFReqLen] dataLen=%d < reqLen=%d\n", dataLen, reqLen)
		return -1
	}
	return int(reqLen)
}

func getHSFRspLen(data []byte) int {
	dataLen := len(data)
	if dataLen < HSF_RSP_HEADER_LEN || data[HSF_MAGIC_IDX] != HSF_MAGIC_TAG || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
		// illegal
		fmt.Printf("[getHSFRspLen] illegal(len=%d): %v\n", dataLen, data)
		return -1
	}
	bodyLen := binary.BigEndian.Uint32(data[16:(16 + 4)])
	rspLen := uint32(HSF_RSP_HEADER_LEN) + bodyLen
	if uint32(dataLen) < rspLen {
		// illegal
		fmt.Printf("[getHSFRspLen] dataLen=%d < reqLen=%d\n", dataLen, rspLen)
		return -1
	}
	return int(rspLen)
}

func (h *rpcHSF) SplitRequest(data []byte) [][]byte {
	var reqs [][]byte
	var start int = 0
	dataLen := len(data)
	for true {
		t := getHSFType(data[start:])
		if t != HSF_TYPE_REQ {
			// invalid data
			fmt.Printf("[SplitRequest] over: get type fail\n")
			break
		}
		hsfDataLen := getHSFReqLen(data[start:])
		if hsfDataLen > 0 && dataLen >= hsfDataLen {
			// there is one valid hsf request
			reqs = append(reqs, data[start:(start+hsfDataLen)])
			start += hsfDataLen
			dataLen -= hsfDataLen
			if dataLen == 0 {
				// finish
				fmt.Printf("[SplitRequest] finish\n")
				break
			}
		} else {
			// invalid data
			fmt.Printf("[SplitRequest] over! reqLen=%d, dataLen=%d\n", hsfDataLen, dataLen)
			break
		}
	}
	return reqs
}

func isValidHSFData(data []byte) bool {
	//return true
	dataLen := len(data)
	if dataLen < 16 || data[HSF_MAGIC_IDX] != HSF_MAGIC_TAG || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
		return false
	}
	var hsfDataLen int
	t := getHSFType(data)
	if t == HSF_TYPE_REQ {
		hsfDataLen = getHSFReqLen(data)
	} else if t == HSF_TYPE_RSP {
		hsfDataLen = getHSFRspLen(data)
	} else {
		return false
	}
	return hsfDataLen > 0 && dataLen >= hsfDataLen
}

func getHSFType(data []byte) int {
	if len(data) <= HSF_TYPE_IDX {
		return -1
	}
	return int(data[HSF_TYPE_IDX])
}

func (h *rpcHSF) GetStreamId(data []byte) string {
	if isValidHSFData(data) == false {
		return ""
	}
	var reqIDRaw []byte
	t := getHSFType(data)
	if t == HSF_TYPE_REQ {
		reqIDRaw = data[HSF_REQ_REQ_ID_IDX:(HSF_REQ_REQ_ID_IDX + HSF_REQ_ID_LEN)]
	} else if t == HSF_TYPE_RSP {
		reqIDRaw = data[HSF_RSP_REQ_ID_IDX:(HSF_RSP_REQ_ID_IDX + HSF_REQ_ID_LEN)]
	} else {
		// illegal data
		return ""
	}
	reqID := binary.BigEndian.Uint64(reqIDRaw)
	reqIDStr := fmt.Sprintf("%d", reqID)
	return reqIDStr
}

func (h *rpcHSF) SetStreamId(data []byte, streamID string) []byte {
	if isValidHSFData(data) == false {
		return data
	}
	reqID, err := strconv.ParseInt(streamID, 10, 64)
	if err != nil {
		return data
	}
	buf := bytes.Buffer{}
	err = binary.Write(&buf, binary.BigEndian, reqID)
	if err != nil {
		return data
	}
	reqIDStr := buf.Bytes()
	reqIDStrLen := len(reqIDStr)
	fmt.Printf("src=%s, len=%d, reqid:%v\n", streamID, reqIDStrLen, reqIDStr)
	var start int
	t := getHSFType(data)
	if t == HSF_TYPE_REQ {
		start = HSF_REQ_REQ_ID_IDX
	} else if t == HSF_TYPE_RSP {
		start = HSF_RSP_REQ_ID_IDX
	} else {
		// illegal data
		return data
	}
	for i := 0; i < HSF_REQ_ID_LEN && i <= reqIDStrLen; i++ {
		data[start+i] = reqIDStr[i]
	}
	return data
}
