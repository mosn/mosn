package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"

	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

type SofaRPCServer struct {
	Listener net.Listener
}

func (s *SofaRPCServer) Run() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				fmt.Printf("[RPC Server] Accept temporary error: %v\n", ne)
				continue
			}
			return //not temporary error, exit
		}
		fmt.Println("[RPC Server] get connection :", conn.RemoteAddr().String())
		go s.Serve(conn)
	}
}

func (s *SofaRPCServer) Serve(conn net.Conn) {
	iobuf := buffer.NewIoBuffer(102400)
	protocol := xprotocol.GetProtocol(bolt.ProtocolName)
	for {
		now := time.Now()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		buf := make([]byte, 10*1024)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				fmt.Printf("[RPC Server] Connect read error: %v\n", err)
				continue
			}

		}
		if bytesRead > 0 {
			iobuf.Write(buf[:bytesRead])
			for iobuf.Len() > 1 {
				cmd, _ := protocol.Decode(nil, iobuf)
				if cmd == nil {
					break
				}
				if req, ok := cmd.(*bolt.Request); ok {
					var iobufresp types.IoBuffer
					var err error
					switch req.CmdCode {
					case bolt.CmdCodeHeartbeat:
						hbAck := protocol.Reply(req.GetRequestId())
						iobufresp, err = protocol.Encode(context.Background(), hbAck)
						fmt.Printf("[RPC Server] reponse heart beat, requestId: %d\n", req.GetRequestId())
					case bolt.CmdCodeRpcRequest:
						resp := buildBoltV1Response(req)
						iobufresp, err = protocol.Encode(context.Background(), resp)
						fmt.Printf("[RPC Server] reponse connection: %s, requestId: %d\n", conn.RemoteAddr().String(), req.GetRequestId())
					}
					if err != nil {
						fmt.Printf("[RPC Server] build response error: %v\n", err)
					} else {
						respdata := iobufresp.Bytes()
						conn.Write(respdata)
					}
				}
			}
		}
	}
}

func buildBoltV1Response(req *bolt.Request) *bolt.Response {
	return &bolt.Response{
		ResponseHeader: bolt.ResponseHeader{
			Protocol:       req.Protocol,
			CmdType:        bolt.CmdTypeResponse,
			CmdCode:        bolt.CmdCodeRpcResponse,
			Version:        req.Version,
			RequestId:      req.RequestId,
			Codec:          req.Codec,
			ResponseStatus: bolt.ResponseStatusSuccess,
		},
	}

}

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	server := &SofaRPCServer{ln}
	server.Run()
}
