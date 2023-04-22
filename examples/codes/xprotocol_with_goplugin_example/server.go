package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"mosn.io/mosn/examples/codes/xprotocol_with_goplugin_example/codec"
)

const respMessage = "Hello, I am server"

type Request struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
}

func decodeRequest(ctx context.Context, data []byte) (cmd interface{}, err error) {
	bytesLen := len(data)

	// 1. least bytes to decode header is RequestHeaderLen
	if bytesLen < codec.RequestHeaderLen {
		return nil, errors.New("short bytesLen")
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(data[codec.RequestPayloadIndex:codec.RequestHeaderLen])
	frameLen := codec.RequestHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return nil, errors.New("not whole bytesLen")
	}

	// 3. Request
	request := &Request{
		Type:       data[codec.TypeIndex],
		RequestId:  binary.BigEndian.Uint32(data[codec.RequestIdIndex : codec.RequestIdEnd+1]),
		PayloadLen: payloadLen,
	}

	//4. copy data for io multiplexing
	request.Payload = data[codec.RequestHeaderLen:]
	return request, err
}

func serve(c net.Conn) {
	reqBuff := make([]byte, 64)

	readLength, err := c.Read(reqBuff)
	if err == nil {
		req := reqBuff[:readLength]

		request, err := decodeRequest(nil, req)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string((request.(*Request).Payload)[:]))

		fmt.Println(c.RemoteAddr())
		bytes := []byte(respMessage)
		buf := make([]byte, 0)
		buf = append(buf, codec.Magic)
		buf = append(buf, codec.TypeMessage)
		buf = append(buf, codec.DirResponse)

		tempBytes := make([]byte, 4)

		binary.BigEndian.PutUint32(tempBytes, request.(*Request).RequestId)
		tempBytesSec := make([]byte, 2)

		binary.BigEndian.PutUint16(tempBytesSec, codec.ResponseStatusSuccess)
		tempBytesThr := make([]byte, 4)

		binary.BigEndian.PutUint32(tempBytesThr, uint32(len(bytes)))
		buf = append(buf, tempBytes...)
		buf = append(buf, tempBytesSec...)
		buf = append(buf, tempBytesThr...)
		buf = append(buf, bytes...)
		c.Write(buf)
		_ = c.Close()

	}
}

func main() {
	//1.create server
	conn, err := net.Listen("tcp", "127.0.0.1:8086")
	if err != nil {
		panic("conn failed")
	}
	for {
		accept := conn.Accept
		c, err := accept()
		if err != nil {
			fmt.Println("accept closed")
		}
		//let serve do accept
		go serve(c)
	}

}
