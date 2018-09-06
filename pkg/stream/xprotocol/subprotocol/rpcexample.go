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

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	Register("rpc-example", &rpcExampleFactory{})
}

type rpcExampleFactory struct{}

func (ref *rpcExampleFactory) CreateSubProtocolCodec(context context.Context) types.Multiplexing {
	return NewRPCExample()
}

type rpcExample struct{}

// NewRPCExample create rpc-example codec
func NewRPCExample() types.Multiplexing {
	return &rpcExample{}
}

var (
	// ReqIDLen rpc-example req id len
	ReqIDLen = 8
	// ReqIDBeginOffset rpc-example req id field offset
	ReqIDBeginOffset = 8
	// ReqDataLen rpc-example request package size
	ReqDataLen = 16
)

func (re *rpcExample) SplitFrame(data []byte) [][]byte {
	var reqs [][]byte
	start := 0
	dataLen := len(data)
	for true {
		if dataLen >= ReqDataLen {
			// there is one valid rpc-example request
			reqs = append(reqs, data[start:(start+ReqDataLen)])
			start += ReqDataLen
			dataLen -= ReqDataLen
			if dataLen == 0 {
				break
			}
		} else {
			// invalid data
			log.DefaultLogger.Tracef("[SplitFrame] over! remain data len = %d", dataLen)
			break
		}
	}
	return reqs
}

func (re *rpcExample) GetStreamID(data []byte) string {
	reqIDRaw := data[8:]
	reqID := binary.BigEndian.Uint64(reqIDRaw)
	reqIDStr := fmt.Sprintf("%d", reqID)
	return reqIDStr
}

func (re *rpcExample) SetStreamID(data []byte, streamID string) []byte {
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
	for i := 0; i < ReqIDLen && i <= reqIDStrLen; i++ {
		data[ReqIDBeginOffset+i] = reqIDStr[i]
	}
	return data
}
