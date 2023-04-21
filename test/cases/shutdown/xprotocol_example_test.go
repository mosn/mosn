//go:build MOSNTest
// +build MOSNTest

package shutdown

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"mosn.io/mosn/examples/codes/xprotocol_with_goplugin_example/codec"
	"mosn.io/mosn/pkg/log"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

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

func sendGoAway(c net.Conn) error {
	buf := make([]byte, 0)
	buf = append(buf, codec.Magic)
	buf = append(buf, codec.TypeGoAway)
	buf = append(buf, codec.DirRequest)

	tempBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytesThr := make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytesThr, 0)
	buf = append(buf, tempBytesThr...)

	if _, err := c.Write(buf); err != nil {
		return err
	}
	return nil
}

func serve(c net.Conn, mode int) error {
	reqBuff := make([]byte, 64)

	readLength, err := c.Read(reqBuff)
	if err == nil {
		if mode == GoAwayBeforeResponse {
			if err := sendGoAway(c); err != nil {
				return err
			}
		}
		req := reqBuff[:readLength]

		request, err := decodeRequest(nil, req)
		if err != nil {
			return err
		}

		bytes := (request.(*Request).Payload)[:]
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

		// sleep 500 ms
		time.Sleep(time.Millisecond * 500)

		if _, err := c.Write(buf); err != nil {
			return err
		}

		if mode == GoAwayAfterResponse {
			// sleep a while to wait the previous request finished in the mosn side.
			time.Sleep(time.Millisecond * 10)

			if err := sendGoAway(c); err != nil {
				return err
			}
		}

		return err
	}

	return err
}

const (
	NoGoAway = iota
	GoAwayBeforeResponse
	GoAwayAfterResponse
)

type Server struct {
	mode      int
	connected int
	closed    int
	conn      net.Listener
}

func startExampleServer(server *Server) {
	//1.create server
	conn, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.DefaultLogger.Errorf("failed to start xprotocol example server: %v", err)
		return
	}
	server.conn = conn
	for {
		accept := conn.Accept
		c, err := accept()
		if err != nil {
			fmt.Println("accept closed")
			break
		}
		server.connected++
		//let serve do accept
		go func() {
			defer c.Close()
			for {
				if err := serve(c, server.mode); err != nil {
					fmt.Printf("serve xprotocol example request failed: %v\n", err)
					break
				}
			}
			server.closed++
		}()
	}
}

func stopExampleServer(server *Server) {
	server.conn.Close()
}

// client
const reqMessage = "Hello World"
const requestId = 1

type Response struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Status     uint16
}

func decode(ctx context.Context, bytes []byte) (cmd interface{}, err error) {
	if dir := bytes[codec.DirIndex]; dir == codec.DirRequest {
		return decodeRequest(ctx, bytes)
	}
	return decodeResponse(ctx, bytes)
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

type Client struct {
	conn   net.Conn
	goaway bool
}

func request(client *Client, msg string) (string, error) {
	bytes := []byte(msg)
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

	// send message
	_, err := client.conn.Write(buf)
	if err != nil {
		return "", fmt.Errorf("write to server failed: %v", err)
	}

	var resp *Response
	var ok bool

	for {
		respBuff := make([]byte, 1024)

		//3.read response
		read, err := client.conn.Read(respBuff)
		if err != nil {
			return "", fmt.Errorf("read from server failed: %v", err)
		}
		data := respBuff[:read]

		//4.decode
		frame, err := decode(nil, data)
		if err != nil {
			return "", fmt.Errorf("decode response failed: %v", err)
		}

		if resp, ok = frame.(*Response); ok {
			break
		}

		if req, ok := frame.(*Request); ok && req.Type == codec.TypeGoAway {
			log.DefaultLogger.Infof("got goaway frame: %v", req)
			client.goaway = true
		}
	}
	return string(resp.Payload[:]), nil
}

// server(mosn) send goaway frame when start graceful stop
func TestXProtocolExampleGracefulStop(t *testing.T) {
	Scenario(t, "xprotocol example graceful stop", func() {
		var m *mosn.MosnOperator
		server := &Server{
			mode: NoGoAway,
		}
		Setup(func() {
			go startExampleServer(server)

			m = mosn.StartMosn(ConfigSimpleXProtocolExample)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			testcases := []struct {
				reqBody string
			}{
				{
					reqBody: "test-req-body",
				},
			}

			for _, tc := range testcases {
				conn, err := net.Dial("tcp", "127.0.0.1:2046")
				if err != nil {
					log.DefaultLogger.Errorf("connect to mosn failed: %v", err)
				}
				defer conn.Close()
				client := &Client{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, false)
				Verify(server.connected, Equal, 1)

				// 2. graceful stop after send request and before received the response
				go func() {
					time.Sleep(time.Millisecond * 100)
					m.GracefulStop()
				}()
				start = time.Now()
				resp, err = request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, true)
				Verify(server.connected, Equal, 1)
			}
		})
		TearDown(func() {
			stopExampleServer(server)
			m.Stop()
		})
	})
}

// client(mosn) got away frame before get response
func TestXProtocolExampleServerGoAwayBeforeResponse(t *testing.T) {
	Scenario(t, "xprotocol example server goaway", func() {
		var m *mosn.MosnOperator
		server := &Server{
			mode: GoAwayBeforeResponse,
		}
		Setup(func() {
			go startExampleServer(server)

			m = mosn.StartMosn(ConfigSimpleXProtocolExample)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			testcases := []struct {
				reqBody string
			}{
				{
					reqBody: "test-req-body",
				},
			}

			for _, tc := range testcases {
				conn, err := net.Dial("tcp", "127.0.0.1:2046")
				if err != nil {
					log.DefaultLogger.Errorf("connect to mosn failed: %v", err)
				}
				defer conn.Close()
				client := &Client{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, false)
				Verify(server.connected, Equal, 1)

				// sleep a while
				time.Sleep(time.Millisecond * 10)
				Verify(server.closed, Equal, 1)

				// 2. try again
				start = time.Now()
				resp, err = request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, false)
				Verify(server.connected, Equal, 2)

				// sleep a while
				time.Sleep(time.Millisecond * 10)
				Verify(server.closed, Equal, 2)
			}
		})
		TearDown(func() {
			stopExampleServer(server)
			m.Stop()
		})
	})
}

// client(mosn) got away frame after get response
func TestXProtocolExampleServerGoAwayAfterResponse(t *testing.T) {
	Scenario(t, "xprotocol example server goaway", func() {
		var m *mosn.MosnOperator
		server := &Server{
			mode: GoAwayAfterResponse,
		}
		Setup(func() {
			go startExampleServer(server)

			m = mosn.StartMosn(ConfigSimpleXProtocolExample)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			testcases := []struct {
				reqBody string
			}{
				{
					reqBody: "test-req-body",
				},
			}

			for _, tc := range testcases {
				conn, err := net.Dial("tcp", "127.0.0.1:2046")
				if err != nil {
					log.DefaultLogger.Errorf("connect to mosn failed: %v", err)
				}
				defer conn.Close()
				client := &Client{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, false)
				Verify(server.connected, Equal, 1)

				// sleep a while
				time.Sleep(time.Millisecond * 100)
				Verify(server.closed, Equal, 1)

				// 2. try again
				start = time.Now()
				resp, err = request(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, false)
				Verify(server.connected, Equal, 2)

				// sleep a while
				time.Sleep(time.Millisecond * 100)
				Verify(server.closed, Equal, 2)
			}
		})
		TearDown(func() {
			stopExampleServer(server)
			m.Stop()
		})
	})
}

const ConfigSimpleXProtocolExample = `{
        "servers":[
                {
                        "default_log_path":"stdout",
                        "default_log_level": "ERROR",
                        "routers": [
                                {
                                        "router_config_name":"router_to_server",
                                        "virtual_hosts":[{
                                                "name":"server_hosts",
                                                "domains": ["*"],
                                                "routers": [
                                                        {
                                                                "route":{"cluster_name":"server_cluster"}
                                                        }
                                                ]
                                        }]
                                }
                        ],
                        "listeners":[
                                {
                                        "address":"127.0.0.1:2046",
                                        "bind_port": true,
                                        "filter_chains": [{
                                                "filters": [
                                                        {
                                                                "type": "proxy",
                                                                "config": {
                                                                        "downstream_protocol": "x_example",
                                                                        "router_config_name":"router_to_server"
                                                                }
                                                        }
                                                ]
                                        }]
                                }
                        ]
                }
        ],
        "cluster_manager":{
                "clusters":[
                        {
                                "name": "server_cluster",
                                "type": "SIMPLE",
                                "lb_type": "LB_RANDOM",
                                "hosts":[
                                        {"address":"127.0.0.1:8080"}
                                ]
                        }
                ]
        },
        "third_part_codec": {
                "codecs": [
                        {
                                "enable": true,
                                "type": "go-plugin",
                                "path": "../codec.so",
                                "loader_func_name": "LoadCodec"
                        }
               ]
       }
}`
