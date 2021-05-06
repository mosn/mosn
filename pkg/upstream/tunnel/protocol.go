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

	RequestType  = []byte{0x01, 0x00}
	ResponseType = []byte{0x02, 0x00}

	typesToFlagAndType = map[string][]byte{
		reflect.TypeOf(ConnectionInitInfo{}).Name(): {0x01, 0x01, 0x00},
	}

	FlagToTypes = map[byte]func() interface{}{
		byte(0x01): func() interface{} {
			return &ConnectionInitInfo{}
		},
	}
)

// ConnectionInitInfo is the basic information of agent host,
// it is sent immediately after the physical connection is established
type ConnectionInitInfo struct {
	ClusterName string `json:"cluster_name"`
	Weight      int64  `json:"weight"`
	HostName    string `json:"host_name"`
}

func WriteBuffer(i interface{}) (buffer.IoBuffer, error) {
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

func Read(buffer api.IoBuffer) interface{} {
	dataBytes := buffer.Bytes()
	if len(dataBytes) == 0 {
		return nil
	}
	if len(dataBytes) <= HeaderLen {
		return nil
	}
	dataLength := binary.BigEndian.Uint16(dataBytes[DataLengthIndex:PayLoadIndex])
	// not enough data
	if len(dataBytes) < HeaderLen+int(dataLength) {
		return nil
	}

	magicBytes := dataBytes[MagicIndex:VersionIndex]
	if bytes.Compare(Magic, magicBytes) != 0 {
		return nil
	}
	_ = dataBytes[VersionIndex:FlagIndex]
	flag := dataBytes[FlagIndex:TypeIndex]
	if f, ok := FlagToTypes[flag[0]]; ok {
		st := f()
		_ = dataBytes[TypeIndex:DataLengthIndex]
		payload := dataBytes[PayLoadIndex : PayLoadIndex+int(dataLength)]
		err := json.Unmarshal(payload, st)
		if err != nil {
			return nil
		}
		buffer.Drain(buffer.Len())
		return st
	}
	return nil
}
