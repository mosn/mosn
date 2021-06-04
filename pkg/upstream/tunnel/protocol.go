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
package tunnel

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

/**
* tunnel protocol
* Request & Response: (byte)
* 0           1           2           3           4           5           6           7           8
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |magic high | magic low |  version  | flag      |    type               |      data length      |
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |                               payload                                                         |
* +-----------------------------------------------------------------------------------------------+
* magic: 0xdabb
*
 */

type ConnectStatus int64

var (
	Magic     = []byte{0x20, 0x88}
	HeaderLen = 8
	Version   = byte(0x01)

	MagicIndex      = 0
	VersionIndex    = 2
	FlagIndex       = 3
	TypeIndex       = 4
	DataLengthIndex = 6
	PayLoadIndex    = 8

	ConnectUnknownFailed   ConnectStatus = 0
	ConnectSuccess         ConnectStatus = 1
	ConnectAuthFailed      ConnectStatus = 2
	ConnectClusterNotExist ConnectStatus = 3

	RequestType  = []byte{0x01, 0x00}
	ResponseType = []byte{0x02, 0x00}

	typesToFlagAndType = map[string][]byte{
		reflect.TypeOf(ConnectionInitInfo{}).Name(): {0x01, 0x01, 0x00},
		reflect.TypeOf(ConnectionInitResponse{}).Name(): {0x02, 0x02, 0x00},
	}

	FlagToTypes = map[byte]reflect.Type{
		byte(0x01): reflect.TypeOf(ConnectionInitInfo{}),
		byte(0x02): reflect.TypeOf(ConnectionInitResponse{}),
	}
)

// ConnectionInitInfo is the basic information of agent host,
// it is sent immediately after the physical connection is established
type ConnectionInitInfo struct {
	ClusterName      string                 `json:"cluster_name"`
	Weight           int64                  `json:"weight"`
	HostName         string                 `json:"host_name"`
	CredentialPolicy string                 `json:"credential_policy"`
	Credential       string                 `json:"credential"`
	Extra            map[string]interface{} `json:"extra"`
}

type ConnectionInitResponse struct {
	Status ConnectStatus `json:"status"`
}

func Encode(i interface{}) (buffer.IoBuffer, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	// alloc encode buffer
	totalLen := HeaderLen + len(data)
	buf := buffer.GetIoBuffer(totalLen)
	// encode header
	buf.Write(Magic)
	buf.WriteByte(Version)
	typeName := reflect.TypeOf(i).Elem().Name()
	if bytes, ok := typesToFlagAndType[typeName]; ok {
		// write flag
		buf.WriteByte(bytes[0])
		// write type
		buf.Write(bytes[1:])
		dataLength := len(data)
		// write data length
		buf.WriteUint16(uint16(dataLength))
		// encode payload
		buf.Write(data)
		return buf, nil
	}
	return nil, errors.New("can't write this struct")
}

func DecodeFromBuffer(buffer api.IoBuffer) (interface{}, error) {
	dataBytes := buffer.Bytes()
	if len(dataBytes) == 0 {
		return nil, nil
	}
	if len(dataBytes) <= HeaderLen {
		return nil, nil
	}
	dataLength := binary.BigEndian.Uint16(dataBytes[DataLengthIndex:PayLoadIndex])
	// not enough data
	if len(dataBytes) < HeaderLen+int(dataLength) {
		return nil, nil
	}

	magicBytes := dataBytes[MagicIndex:VersionIndex]
	if !bytes.Equal(Magic, magicBytes) {
		return nil, fmt.Errorf("magic value is not same")
	}
	_ = dataBytes[VersionIndex:FlagIndex]
	flag := dataBytes[FlagIndex:TypeIndex]
	if t, ok := FlagToTypes[flag[0]]; ok {
		st := reflect.New(t)
		_ = dataBytes[TypeIndex:DataLengthIndex]
		payload := dataBytes[PayLoadIndex : PayLoadIndex+int(dataLength)]
		err := json.Unmarshal(payload, st.Interface())
		if err != nil {
			return nil, err
		}
		buffer.Drain(buffer.Len())
		return st.Elem().Interface(), nil
	}
	return nil, nil
}
