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

package main

import (
	"context"
	"encoding/binary"
	"fmt"

	"mosn.io/pkg/buffer"
)

///////////////////////////////////////////////////////////////////////
// Thrift protocol decoder, ported from thrift-lib-go
//
// See: https://github.com/apache/thrift/tree/master/lib/go
//
// NOTE: Not finished yet, only for ICBC POC now
// Supported data structure in field:
//	- int32
//	- object (class)
///////////////////////////////////////////////////////////////////////

const (
	FieldTypeBytesUsed int32 = 1
	FieldIdBytesUsed   int32 = 2
)

type TMessageType int32

const (
	TmessageType TMessageType = 0
	CALL         TMessageType = 1
	REPLY        TMessageType = 2
	EXCEPTION    TMessageType = 3
	ONEWAY       TMessageType = 4

	VersionMask           = 0xffff0000
	Version1              = 0x80010000
	DefaultRecursionDepth = 64
)

// Type constants in the Thrift protocol
type TType byte

const (
	STOP   = 0
	VOID   = 1
	BOOL   = 2
	BYTE   = 3
	I08    = 3
	DOUBLE = 4
	I16    = 6
	I32    = 8
	I64    = 10
	STRING = 11
	UTF7   = 11
	STRUCT = 12
	MAP    = 13
	SET    = 14
	LIST   = 15
	UTF8   = 16
	UTF16  = 17
	//BINARY = 18   wrong and unusued
)

const readLimit = 32768

func readI32(buff []byte) (value int32, err error) {
	if len(buff) < 4 {
		return 0, fmt.Errorf("readI32 failed: buff len<4: %+v", len(buff))
	}
	value = int32(binary.BigEndian.Uint32(buff[:4]))
	return value, err
}

func readI16(buff []byte) (value int16, err error) {
	if len(buff) < 2 {
		return 0, fmt.Errorf("readI16 failed: buff len<2: %+v", len(buff))
	}
	value = int16(binary.BigEndian.Uint16(buff[:2]))
	return value, err
}

func readByte(buff []byte) (value int8, err error) {
	if len(buff) < 1 {
		return 0, fmt.Errorf("readByte failed: buff len<1: %+v", len(buff))
	}
	value = int8(buff[0])
	return value, err
}

func readString(buff []byte) (value string, StringLen int32, err error) {
	// read string length first, in fixed headed 4 bytes
	size, e := readI32(buff)
	if e != nil {
		return "", 0, fmt.Errorf("readString failed: buff len<4: %+v", len(buff))
	}
	if size < 0 {
		return "", 0, fmt.Errorf("readString failed: string size: %+v", size)
	}

	// read string body start from the 5th byte
	s, e := readStringBody(buff[4:], size)
	return s, size, e
}

func readStringBody(buff []byte, size int32) (value string, err error) {
	if len(buff) < int(size) {
		return "", fmt.Errorf("readStringBody failed, wanted string len: %+v, but got: %+v", size, len(buff))
	}

	return string(buff[:size]), nil
}

func readMessage(buff []byte) (name string, typeId TMessageType, seqId, messageLen int32, err error) {
	size, e := readI32(buff)
	if e != nil {
		return "", 0, 0, 0, e
	}
	messageLen += 4

	if size < 0 {
		var strLen int32

		typeId = TMessageType(size & 0x0ff)
		version := int64(size) & VersionMask
		if version != Version1 {
			return name, typeId, seqId, messageLen, fmt.Errorf("bad version in message")
		}
		name, strLen, e = readString(buff[4:])
		if e != nil {
			return name, typeId, seqId, messageLen, e
		}
		messageLen += 4 + strLen
		seqId, e = readI32(buff[4+4+strLen:])
		if e != nil {
			return name, typeId, seqId, messageLen, e
		}
		messageLen += 4
	}

	return name, typeId, seqId, messageLen, nil
}

func readFieldBegin(buffer []byte) (name string, typeId TType, seqId int16, err error) {
	t, err := readByte(buffer)
	if err != nil {
		return name, typeId, seqId, err
	}
	typeId = TType(t)

	if t != STOP {
		seqId, err = readI16(buffer)
	}
	return name, typeId, seqId, err
}

func readFieldEnd(buffer []byte) (name string, typeId TType, seqId int16, err error) {
	return "", 0, 0, nil
}

func readFieldType(buffer []byte) (typeId TType, err error) {
	t, err := readByte(buffer)
	if err != nil {
		return 0, err
	}

	return TType(t), err
}

func eatField(buff []byte, fieldType TType, maxDepth int) (fieldLen int32, err error) {
	if maxDepth <= 0 {
		return 0, fmt.Errorf("depth limit exceeded")
	}

	switch fieldType {
	case BOOL:
		// todo
		//_, err = self.ReadBool(ctx)
		return
	case BYTE:
		_, err = readByte(buff)
		if err != nil {
			return 0, err
		}
		return 1, nil
	case I16:
		_, err = readI16(buff)
		if err != nil {
			return 0, err
		}
		return 2, nil
	case I32:
		_, err = readI32(buff)
		if err != nil {
			return 0, err
		}
		return 4, nil
	case I64:
		// todo
		//_, err = readI16(buff)
		//if err != nil {
		//	return 0, err
		//}
		//*curFieldLen += 2
		//return 2, nil
	case DOUBLE:
		// todo
		//_, err = readI16(buff)
		//if err != nil {
		//	return 0, err
		//}
		//*curFieldLen += 2
		//return 2, nil
	case STRING:
		_, strLen, err := readString(buff)
		if err != nil {
			return 0, err
		}
		return 4 + strLen, nil
	case STRUCT:
		for {
			_, typeId, _, err := readFieldBegin(buff[fieldLen:])
			if err != nil {
				return 0, err
			}

			fieldLen += FieldTypeBytesUsed
			if typeId == STOP {
				break
			}

			fieldLen += FieldIdBytesUsed

			thisFieldLen, err := eatField(buff[fieldLen:], typeId, maxDepth-1)
			if err != nil {
				return 0, err
			}

			fieldLen += thisFieldLen

			// fixme @lude
			_, _, _, err = readFieldEnd(buff)
			if err != nil {
				return 0, err
			}
		}

		return fieldLen, nil
	case MAP:
		// todo
		//keyType, valueType, size, err := self.ReadMapBegin(ctx)
		//if err != nil {
		//	return err
		//}
		//for i := 0; i < size; i++ {
		//	err := Skip(ctx, self, keyType, maxDepth-1)
		//	if err != nil {
		//		return err
		//	}
		//	self.Skip(ctx, valueType)
		//}
		//return self.ReadMapEnd(ctx)
	case SET:
		// todo
		//elemType, size, err := self.ReadSetBegin(ctx)
		//if err != nil {
		//	return err
		//}
		//for i := 0; i < size; i++ {
		//	err := Skip(ctx, self, elemType, maxDepth-1)
		//	if err != nil {
		//		return err
		//	}
		//}
		//return self.ReadSetEnd(ctx)
	case LIST:
		// todo
		//elemType, size, err := self.ReadListBegin(ctx)
		//if err != nil {
		//	return err
		//}
		//for i := 0; i < size; i++ {
		//	err := Skip(ctx, self, elemType, maxDepth-1)
		//	if err != nil {
		//		return err
		//	}
		//}
		//return self.ReadListEnd(ctx)
	default:
		return 0, fmt.Errorf("unknown data type %d", fieldType)
	}

	return 0, fmt.Errorf("should not be here")
}

func decodeDataBytes(data []byte) (int32, int32, string, error) {
	var frameLen int32

	name, _, _, msgLen, err := readMessage(data)
	if err != nil {
		return 0, 0, "", err
	}

	frameLen += msgLen

	for {
		fieldType, err := readFieldType(data[frameLen:])
		if err != nil {
			return 0, 0, "", nil
		}

		frameLen += FieldTypeBytesUsed
		if fieldType == STOP {
			break
		}

		frameLen += FieldIdBytesUsed

		thisFieldLen, err := eatField(data[frameLen:], fieldType, DefaultRecursionDepth)
		if err != nil {
			return 0, 0, "", err
		}

		frameLen += thisFieldLen
	}

	return msgLen, frameLen, name, err
}

func decodeData(data buffer.IoBuffer) (int32, int32, string, error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	if bytesLen < MessageMinLen {
		return 0, 0, "", fmt.Errorf("no enough data")
	}

	return decodeDataBytes(bytes)
}

func decodeRequest(ctx context.Context, data buffer.IoBuffer, oneway bool, message Message) (cmd interface{}, err error) {
	//fmt.Printf("====================decodeRequest==============\n")
	logger.Printf("in decodeRequest")

	msgLen, frameLen, name, err := decodeData(data)
	if err != nil {
		return nil, err
	}

	buf := bufferByContext(ctx)
	request := &buf.request

	cmdType := MessageTypeCall
	if oneway {
		cmdType = MessageTypeOneway
	}

	request.RequestHeader = RequestHeader{
		Protocol: ProtocolCode, // fixme @lude
		CmdType:  byte(cmdType),
		//CmdCode:  CmdCodeRequest,
		//CmdCode:   binary.BigEndian.Uint16(bytes[2:4]),
		Version:   message.Version,
		RequestId: uint32(message.SeqId),
		//Codec:     bytes[9],
		//Timeout:    int32(binary.BigEndian.Uint32(bytes[10:14])),
		//ClassLen:   classLen,
		HeaderLen:  uint16(msgLen),
		ContentLen: uint32(frameLen) - uint32(msgLen),
	}

	if name == "ping" {
		request.CmdCode = CmdCodeHeartbeat
	} else {
		request.CmdCode = CmdCodeRequest
	}

	request.Data = buffer.GetIoBuffer(int(frameLen))

	request.Data.Write(data.Bytes()[:frameLen])
	request.rawData = request.Data.Bytes()

	contentIndex := msgLen

	request.BytesHeader.Set("X-Govern-Service", fmt.Sprintf("%s@thrift", message.Name))
	request.BytesHeader.Set("X-Govern-Method", message.Name)

	request.BytesHeader.Set("service", fmt.Sprintf("%s@thrift", message.Name))

	if contentIndex > 0 {
		request.rawContent = request.rawData[contentIndex:frameLen]
		request.Content = buffer.NewIoBufferBytes(request.rawContent)
	}

	data.Drain(int(frameLen))

	return request, err
}

func decodeResponse(ctx context.Context, data buffer.IoBuffer, message Message) (cmd interface{}, err error) {
	logger.Printf("in decodeResponse")

	msgLen, frameLen, _, err := decodeData(data)
	if err != nil {
		return nil, err
	}

	buf := bufferByContext(ctx)
	response := &buf.response

	cmdType := MessageTypeReply

	response.ResponseHeader = ResponseHeader{
		Protocol: ProtocolCode, // fixme @lude
		CmdType:  byte(cmdType),
		CmdCode:  CmdCodeResponse,
		//CmdCode:   binary.BigEndian.Uint16(bytes[2:4]),
		Version:   message.Version,
		RequestId: uint32(message.SeqId),
		//Codec:     bytes[9],
		//Timeout:    int32(binary.BigEndian.Uint32(bytes[10:14])),
		//ClassLen:   classLen,
		HeaderLen:  uint16(msgLen),
		ContentLen: uint32(frameLen) - uint32(msgLen),
	}

	response.Data = buffer.GetIoBuffer(int(frameLen))

	response.Data.Write(data.Bytes()[:frameLen])
	response.rawData = response.Data.Bytes()

	contentIndex := msgLen

	response.BytesHeader.Set("X-Govern-Service", fmt.Sprintf("%s@thrift", message.Name))
	response.BytesHeader.Set("X-Govern-Method", message.Name)

	response.BytesHeader.Set("service", fmt.Sprintf("%s@thrift", message.Name))

	if contentIndex > 0 {
		response.rawContent = response.rawData[contentIndex:frameLen]
		response.Content = buffer.NewIoBufferBytes(response.rawContent)
	}

	data.Drain(int(frameLen))

	return response, err
}
