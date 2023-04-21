package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"mosn.io/mosn/examples/codes/xprotocol_with_goplugin_example/codec"
)

const reqMessage = "Hello World"
const requestId = 1

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
	if bytesLen < codec.ResponseHeaderLen {
		return nil, errors.New("bytesLen<ResponseHeaderLen")
	}

	payloadLen := binary.BigEndian.Uint32(bytes[codec.ResponsePayloadIndex:codec.ResponseHeaderLen])

	//2.total protocol length
	frameLen := codec.ResponseHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return nil, errors.New("short bytesLen")
	}

	// 3.  response
	response := &Response{

		Type:       bytes[codec.DirIndex],
		RequestId:  binary.BigEndian.Uint32(bytes[codec.RequestIdIndex : codec.RequestIdEnd+1]),
		PayloadLen: payloadLen,
		Status:     codec.ResponseStatusSuccess,
	}

	//4. copy data for io multiplexing
	response.Payload = bytes[codec.ResponseHeaderLen:]
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

	buf = append(buf, codec.Magic)
	buf = append(buf, codec.TypeMessage)
	buf = append(buf, codec.DirRequest)
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
		panic(err)
	}
	resp := respBuff[:read]

	//4.decodeResponse
	response, err := decodeResponse(nil, resp)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(response.(*Response).Payload[:]))

}
