package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

const reqMessage = "Hello World"
const requestId = 1

const (
	ProtocolName string = "example" // protocol

	Magic       byte = 'x' //magic
	MagicIdx         = 0   //magicIndex
	DirRequest  byte = 0   // dir
	DirResponse byte = 1   // dir

	TypeHeartbeat byte = 0 // cmd code
	TypeMessage   byte = 1
	TypeGoAway    byte = 2

	ResponseStatusSuccess uint16 = 0 // 0x00 response status
	ResponseStatusError   uint16 = 1 // 0x01

	RequestHeaderLen  int = 11 // protocol header fields length
	ResponseHeaderLen int = 13
	MinimalDecodeLen  int = RequestHeaderLen // minimal length for decoding

	RequestIdIndex       = 3
	DirIndex             = 1
	RequestPayloadIndex  = 7
	RequestIdEnd         = 6
	ResponsePayloadIndex = 9
)

type Response struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Status     uint16
}

func decodeResponse(ctx context.Context, bytes []byte) (cmd interface{}, err error) {
	bytesLen := len(bytes)

	// 1. least bytes to decode header is ResponseHeaderLen
	if bytesLen < ResponseHeaderLen {
		return nil, errors.New("bytesLen<ResponseHeaderLen")
	}

	payloadLen := binary.BigEndian.Uint32(bytes[ResponsePayloadIndex:ResponseHeaderLen])

	//2.total protocol length
	frameLen := ResponseHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return nil, errors.New("short bytesLen")
	}

	// 3.  response
	response := &Response{

		Type:       bytes[DirIndex],
		RequestId:  binary.BigEndian.Uint32(bytes[RequestIdIndex : RequestIdEnd+1]),
		PayloadLen: payloadLen,
		Status:     ResponseStatusSuccess,
	}

	//4. copy data for io multiplexing
	response.Payload = bytes[ResponseHeaderLen:]
	return response, nil
}

func main() {

	//1.create client
	conn, err := net.Dial("tcp", "127.0.0.1:2045")
	if err != nil {
		panic("conn failed")
	}
	bytes := []byte(reqMessage)
	buf := make([]byte, 0)

	buf = append(buf, Magic)
	buf = append(buf, TypeMessage)
	buf = append(buf, DirRequest)
	tempBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, requestId)
	tempBytesSec := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytesSec, uint32(len(bytes)))
	buf = append(buf, tempBytes...)
	buf = append(buf, tempBytesSec...)
	buf = append(buf, bytes...)

	//2.send message
	_, err = conn.Write(buf)
	if err != nil {
		panic("write failed")
	}

	respBuff := make([]byte, 1024)

	//3.read response
	read, err := conn.Read(respBuff)
	if err != nil {
		fmt.Println(err.Error())
		panic("read failed")
	}
	resp := respBuff[:read]

	//4.decodeResponse
	response, err := decodeResponse(nil, resp)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(response.(*Response).Payload[:]), "------resp")

}
