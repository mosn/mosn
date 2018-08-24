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

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/log"
)

func init() {
	Register("X-hsf", &pluginHSFFactory{})
}

type pluginHSFFactory struct{}

func (ref *pluginHSFFactory) CreateSubProtocolCodec(context context.Context) types.Multiplexing {
	return NewRPCHSF()
}

type rpcHSF struct{}

func NewRPCHSF() types.Tracing {
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
 * magic: 0x0e
 *
 * Heartbeat:
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |magic| type| ver |  externds bytes |        requst id                              | timeout   |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |  timeout  |
 * +-----------+
 * hb-magic: 0x0c
 */

const (
	HSF_REQ_HEADER_LEN   = 35
	HSF_RSP_HEADER_LEN   = 20
	HSF_MAGIC_IDX        = 0
	HSF_VERSION_IDX      = 1
	HSF_TYPE_IDX         = 2
	HSF_TYPE_REQ         = 0
	HSF_TYPE_RSP         = 1
	HSF_REQ_REQ_ID_IDX   = 7
	HSF_REQ_SVC_NAME_IDX = 19
	HSF_REQ_MTD_NAME_IDX = 23
	HSF_RSP_REQ_ID_IDX   = 8
	HSF_REQ_ID_LEN       = 8

	HSF_MAGIC_TAG  = 0x0e
	HSF_VERSION_V1 = 1

	HSF_HB_MAGIC_TAG  = 0x0c
	HSF_HB_HEADER_LEN = 18
	HSF_HB_REQ_ID_IDX = 6
)

type hsfReqAttr struct {
	headerLen      uint32
	serviceNameLen uint32
	methodNameLen  uint32
	paraTypeNum    uint32
	paraTypeLen    uint32
	argLen         uint32
	propsLen       uint32
}

func parseHSFReq(data []byte) (int, *hsfReqAttr) {
	dataLen := len(data)
	if dataLen < HSF_REQ_HEADER_LEN || data[HSF_MAGIC_IDX] != HSF_MAGIC_TAG || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
		// illegal
		log.DefaultLogger.Tracef("[parseHSFReq] illegal(len=%d): %v\n", dataLen, data)
		return -1, nil
	}
	attr := &hsfReqAttr{}
	attr.serviceNameLen = binary.BigEndian.Uint32(data[HSF_REQ_SVC_NAME_IDX:(HSF_REQ_SVC_NAME_IDX + 4)])
	attr.methodNameLen = binary.BigEndian.Uint32(data[HSF_REQ_MTD_NAME_IDX:(HSF_REQ_MTD_NAME_IDX + 4)])
	attr.paraTypeNum = binary.BigEndian.Uint32(data[27:(27 + 4)])
	attr.headerLen = uint32(HSF_REQ_HEADER_LEN)
	idx := 31
	attr.paraTypeLen = uint32(0)
	for i := uint32(0); i < attr.paraTypeNum; i++ {
		l := binary.BigEndian.Uint32(data[idx:(idx + 4)])
		idx += 4
		attr.paraTypeLen += l
		attr.headerLen += 4
	}
	attr.argLen = uint32(0)
	for i := uint32(0); i < attr.paraTypeNum; i++ {
		l := binary.BigEndian.Uint32(data[idx:(idx + 4)])
		idx += 4
		attr.argLen += l
		attr.headerLen += 4
	}
	attr.propsLen = binary.BigEndian.Uint32(data[idx:(idx + 4)])
	reqLen := attr.headerLen + attr.serviceNameLen + attr.methodNameLen + attr.paraTypeLen + attr.argLen + attr.propsLen
	if uint32(dataLen) < reqLen {
		// illegal
		log.DefaultLogger.Tracef("[parseHSFReq] dataLen=%d < reqLen=%d\n", dataLen, reqLen)
		return -1, nil
	}
	return int(reqLen), attr
}

func getHSFReqLen(data []byte) int {
	len, _ := parseHSFReq(data)
	return len
}

func getHSFRspLen(data []byte) int {
	dataLen := len(data)
	if dataLen < HSF_RSP_HEADER_LEN || data[HSF_MAGIC_IDX] != HSF_MAGIC_TAG || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
		// illegal
		log.DefaultLogger.Tracef("[getHSFRspLen] illegal(len=%d): %v\n", dataLen, data)
		return -1
	}
	bodyLen := binary.BigEndian.Uint32(data[16:(16 + 4)])
	rspLen := uint32(HSF_RSP_HEADER_LEN) + bodyLen
	if uint32(dataLen) < rspLen {
		// illegal
		log.DefaultLogger.Tracef("[getHSFRspLen] dataLen=%d < reqLen=%d\n", dataLen, rspLen)
		return -1
	}
	return int(rspLen)
}

func isHSFHb(data []byte) bool {
	if len(data) < HSF_HB_HEADER_LEN {
		return false
	}
	return data[0] == HSF_HB_MAGIC_TAG
}

func (h *rpcHSF) SplitFrame(data []byte) [][]byte {
	var reqs [][]byte
	var start int = 0
	dataLen := len(data)
	for true {
		var hsfDataLen int
		if isHSFHb(data[start:]) {
			// heart-beat msg(fixed length)
			hsfDataLen = HSF_HB_HEADER_LEN
		} else {
			t := getHSFType(data[start:])
			if t == HSF_TYPE_REQ {
				hsfDataLen = getHSFReqLen(data[start:])
			} else if t == HSF_TYPE_RSP {
				hsfDataLen = getHSFRspLen(data[start:])
			} else {
				// invalid data
				log.DefaultLogger.Tracef("[SplitFrame] over: type(%d) isn't request. req_cnt=%d\n", t, len(reqs))
				break
			}
		}
		if hsfDataLen > 0 && dataLen >= hsfDataLen {
			// there is one valid hsf request
			reqs = append(reqs, data[start:(start+hsfDataLen)])
			start += hsfDataLen
			dataLen -= hsfDataLen
			if dataLen == 0 {
				// finish
				//log.DefaultLogger.Tracef("[SplitFrame] finish\n")
				break
			}
		} else {
			// invalid data
			log.DefaultLogger.Tracef("[SplitFrame] over! reqLen=%d, dataLen=%d. req_cnt=%d\n", hsfDataLen, dataLen, len(reqs))
			break
		}
	}
	return reqs
}

func isValidHSFData(data []byte) bool {
	//return true
	dataLen := len(data)
	if data[HSF_MAGIC_IDX] == HSF_MAGIC_TAG {
		if dataLen < 16 || data[HSF_VERSION_IDX] != HSF_VERSION_V1 {
			return false
		}
	} else if data[HSF_MAGIC_IDX] == HSF_HB_MAGIC_TAG {
		if dataLen < HSF_HB_HEADER_LEN {
			return false
		}
		return true
	} else {
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
	if data[HSF_MAGIC_IDX] == HSF_HB_MAGIC_TAG {
		reqIDRaw = data[HSF_HB_REQ_ID_IDX:(HSF_HB_REQ_ID_IDX + HSF_REQ_ID_LEN)]
	} else {
		t := getHSFType(data)
		if t == HSF_TYPE_REQ {
			reqIDRaw = data[HSF_REQ_REQ_ID_IDX:(HSF_REQ_REQ_ID_IDX + HSF_REQ_ID_LEN)]
		} else if t == HSF_TYPE_RSP {
			reqIDRaw = data[HSF_RSP_REQ_ID_IDX:(HSF_RSP_REQ_ID_IDX + HSF_REQ_ID_LEN)]
		} else {
			// illegal data
			return ""
		}
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
	log.DefaultLogger.Tracef("src=%s, len=%d, reqid:%v\n", streamID, reqIDStrLen, reqIDStr)

	var start int
	if data[HSF_MAGIC_IDX] == HSF_HB_MAGIC_TAG {
		start = HSF_HB_REQ_ID_IDX
	} else {
		t := getHSFType(data)
		if t == HSF_TYPE_REQ {
			start = HSF_REQ_REQ_ID_IDX
		} else if t == HSF_TYPE_RSP {
			start = HSF_RSP_REQ_ID_IDX
		} else {
			// illegal data
			return data
		}
	}

	for i := 0; i < HSF_REQ_ID_LEN && i <= reqIDStrLen; i++ {
		data[start+i] = reqIDStr[i]
	}
	return data
}

func (h *rpcHSF) GetServiceName(data []byte) string {
	t := getHSFType(data)
	if t != HSF_TYPE_REQ {
		return ""
	}
	l, attr := parseHSFReq(data)
	if l <= 0 {
		return ""
	}
	idx := attr.headerLen
	return string(data[idx:(idx + attr.serviceNameLen)])
}

func (h *rpcHSF) GetMethodName(data []byte) string {
	t := getHSFType(data)
	if t != HSF_TYPE_REQ {
		return ""
	}
	l, attr := parseHSFReq(data)
	if l <= 0 {
		return ""
	}
	idx := attr.headerLen + attr.serviceNameLen
	return string(data[idx:(idx + attr.methodNameLen)])
}
