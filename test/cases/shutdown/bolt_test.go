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

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
	"mosn.io/pkg/header"
)

type BoltRequest struct {
	BoltRequestHeader

	RawData    []byte // raw data
	RawMeta    []byte // sub slice of raw data, start from protocol code, ends to content length
	RawClass   []byte // sub slice of raw data, class bytes
	RawHeader  []byte // sub slice of raw data, header bytes
	RawContent []byte // sub slice of raw data, content bytes
}

type BoltRequestHeader struct {
	Protocol   byte // meta fields
	CmdType    byte
	CmdCode    uint16
	Version    byte
	RequestId  uint32
	Codec      byte
	Timeout    int32
	ClassLen   uint16
	HeaderLen  uint16
	ContentLen uint32

	Class string // payload fields
	header.BytesHeader
}

func decodeBoltRequest(ctx context.Context, data []byte) (cmd interface{}, err error) {
	dataLen := len(data)

	// 1. least bytes to decode header is RequestHeaderLen(22)
	if dataLen < bolt.RequestHeaderLen {
		return nil, errors.New("data length is less than bolt request header length")
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(data[14:16])
	headerLen := binary.BigEndian.Uint16(data[16:18])
	contentLen := binary.BigEndian.Uint32(data[18:22])

	frameLen := bolt.RequestHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	if dataLen < frameLen {
		return nil, errors.New("data length is less than bolt request frame length")
	}

	// 3. BoltRequest
	request := &BoltRequest{
		BoltRequestHeader: BoltRequestHeader{
			Protocol:   bolt.ProtocolCode,
			CmdType:    bolt.CmdTypeRequest,
			CmdCode:    binary.BigEndian.Uint16(data[2:4]),
			Version:    data[4],
			RequestId:  binary.BigEndian.Uint32(data[5:9]),
			Codec:      data[9],
			Timeout:    int32(binary.BigEndian.Uint32(data[10:14])),
			ClassLen:   classLen,
			HeaderLen:  headerLen,
			ContentLen: contentLen,
		},
	}

	//4. copy data for io multiplexing
	request.RawData = data[:frameLen]

	//5. process wrappers: Class, Header, Content, Data
	headerIndex := bolt.RequestHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	request.RawMeta = request.RawData[:bolt.RequestHeaderLen]
	if classLen > 0 {
		request.RawClass = request.RawData[bolt.RequestHeaderLen:headerIndex]
		request.Class = string(request.RawClass)
	}
	if headerLen > 0 {
		request.RawHeader = request.RawData[headerIndex:contentIndex]
		err = header.DecodeHeader(request.RawHeader, &request.BytesHeader)
	}
	if contentLen > 0 {
		request.RawContent = request.RawData[contentIndex:]
	}
	return request, err
}

func sendBoltGoAway(c net.Conn) error {
	buf := make([]byte, 0)
	buf = append(buf, bolt.ProtocolCode)
	buf = append(buf, bolt.CmdTypeRequest)

	tempBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, bolt.CmdCodeGoAway)
	buf = append(buf, tempBytes...)

	buf = append(buf, bolt.ProtocolVersion)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 0)
	buf = append(buf, tempBytes...)

	buf = append(buf, bolt.Hessian2Serialize)

	tempBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 0)
	buf = append(buf, tempBytes...)

	if _, err := c.Write(buf); err != nil {
		return err
	}
	return nil
}

func boltServe(c net.Conn, mode int) error {
	reqBuff := make([]byte, 64)

	readLength, err := c.Read(reqBuff)
	if err == nil {
		if mode == GoAwayBeforeResponse {
			if err := sendBoltGoAway(c); err != nil {
				return err
			}
		}
		req := reqBuff[:readLength]

		request, err := decodeBoltRequest(nil, req)
		if err != nil {
			return err
		}

		bytes := (request.(*BoltRequest).RawContent)[:]
		buf := make([]byte, 0)
		buf = append(buf, bolt.ProtocolCode)
		buf = append(buf, bolt.CmdTypeResponse)

		tempBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(tempBytes, bolt.CmdCodeRpcResponse)
		buf = append(buf, tempBytes...)

		buf = append(buf, bolt.ProtocolVersion)

		tempBytes = make([]byte, 4)
		binary.BigEndian.PutUint32(tempBytes, request.(*BoltRequest).BoltRequestHeader.RequestId)
		buf = append(buf, tempBytes...)

		buf = append(buf, bolt.Hessian2Serialize)

		tempBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(tempBytes, bolt.ResponseStatusSuccess)
		buf = append(buf, tempBytes...)

		tempBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(tempBytes, 0)
		buf = append(buf, tempBytes...)

		tempBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(tempBytes, 0)
		buf = append(buf, tempBytes...)

		tempBytes = make([]byte, 4)
		binary.BigEndian.PutUint32(tempBytes, uint32(len(bytes)))
		buf = append(buf, tempBytes...)

		buf = append(buf, bytes...)

		// sleep 500 ms
		time.Sleep(time.Millisecond * 500)

		if _, err := c.Write(buf); err != nil {
			return err
		}

		if mode == GoAwayAfterResponse {
			// sleep a while to wait the previous request finished in the mosn side.
			time.Sleep(time.Millisecond * 10)

			if err := sendBoltGoAway(c); err != nil {
				return err
			}
		}

		return err
	}

	return err
}

type BoltServer struct {
	mode      int
	connected int
	closed    int
	conn      net.Listener
}

func startBoltServer(server *BoltServer) {
	// create server
	conn, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.DefaultLogger.Errorf("failed to start bolt server: %v", err)
		return
	}
	server.conn = conn
	for {
		c, err := conn.Accept()
		if err != nil {
			fmt.Printf("accept error: %v\n", err)
			break
		}
		server.connected++
		// let serve do accept
		go func() {
			defer c.Close()
			for {
				if err := boltServe(c, server.mode); err != nil {
					fmt.Printf("serve bolt request failed: %v\n", err)
					break
				}
			}
			server.closed++
		}()
	}
}

func stopBoltServer(server *BoltServer) {
	server.conn.Close()
}

type BoltResponse struct {
	BoltResponseHeader

	RawData    []byte // raw data
	RawMeta    []byte // sub slice of raw data, start from protocol code, ends to content length
	RawClass   []byte // sub slice of raw data, class bytes
	RawHeader  []byte // sub slice of raw data, header bytes
	RawContent []byte // sub slice of raw data, content bytes
}

type BoltResponseHeader struct {
	Protocol       byte // meta fields
	CmdType        byte
	CmdCode        uint16
	Version        byte
	RequestId      uint32
	Codec          byte
	ResponseStatus uint16
	ClassLen       uint16
	HeaderLen      uint16
	ContentLen     uint32

	Class string // payload fields
	header.BytesHeader
}

func boltDecode(ctx context.Context, bytes []byte) (cmd interface{}, err error) {
	if len(bytes) >= bolt.LessLen {
		cmdType := bytes[1]
		switch cmdType {
		case bolt.CmdTypeRequest:
			return decodeBoltRequest(ctx, bytes)
		case bolt.CmdTypeResponse:
			return decodeBoltResponse(ctx, bytes)
		}
	}
	return nil, errors.New("bytesLen is less than bolt.LessLen")
}

func decodeBoltResponse(ctx context.Context, data []byte) (cmd interface{}, err error) {
	dataLen := len(data)

	// 1. least bytes to decode header is ResponseHeaderLen
	if dataLen < bolt.LessLen {
		return nil, errors.New("dataLen is less than bolt.LessLen")
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(data[12:14])
	headerLen := binary.BigEndian.Uint16(data[14:16])
	contentLen := binary.BigEndian.Uint32(data[16:20])

	frameLen := bolt.ResponseHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	if dataLen < frameLen {
		return nil, errors.New("dataLen is less than frameLen")
	}

	// 3.  decode header
	response := &BoltResponse{
		BoltResponseHeader: BoltResponseHeader{
			Protocol:       bolt.ProtocolCode,
			CmdType:        bolt.CmdTypeResponse,
			CmdCode:        binary.BigEndian.Uint16(data[2:4]),
			Version:        data[4],
			RequestId:      binary.BigEndian.Uint32(data[5:9]),
			Codec:          data[9],
			ResponseStatus: binary.BigEndian.Uint16(data[10:12]),
			ClassLen:       classLen,
			HeaderLen:      headerLen,
			ContentLen:     contentLen,
		}}

	//4. copy data for io multiplexing
	response.RawData = data[:frameLen]

	//5. process wrappers: Class, Header, Content, Data
	headerIndex := bolt.ResponseHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	response.RawMeta = response.RawData[:bolt.ResponseHeaderLen]
	if classLen > 0 {
		response.RawClass = response.RawData[bolt.ResponseHeaderLen:headerIndex]
		response.Class = string(response.RawClass)
	}
	if headerLen > 0 {
		response.RawHeader = response.RawData[headerIndex:contentIndex]
		err = header.DecodeHeader(response.RawHeader, &response.BytesHeader)
	}
	if contentLen > 0 {
		response.RawContent = response.RawData[contentIndex:]
	}
	return response, err
}

type BoltClient struct {
	conn   net.Conn
	goaway bool
}

func boltRequest(client *BoltClient, msg string) (string, error) {
	bytes := []byte(msg)
	buf := make([]byte, 0)
	buf = append(buf, bolt.ProtocolCode)
	buf = append(buf, bolt.CmdTypeRequest)

	tempBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, bolt.CmdCodeRpcRequest)
	buf = append(buf, tempBytes...)

	buf = append(buf, bolt.ProtocolVersion)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 1)
	buf = append(buf, tempBytes...)

	buf = append(buf, bolt.Hessian2Serialize)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(tempBytes, 0)
	buf = append(buf, tempBytes...)

	tempBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(tempBytes, uint32(len(bytes)))
	buf = append(buf, tempBytes...)

	buf = append(buf, bytes...)

	// send message
	_, err := client.conn.Write(buf)
	if err != nil {
		return "", fmt.Errorf("write to server failed: %v", err)
	}

	var resp *BoltResponse
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
		frame, err := boltDecode(nil, data)
		if err != nil {
			return "", fmt.Errorf("decode response failed: %v", err)
		}

		if req, ok := frame.(*BoltRequest); ok && req.BoltRequestHeader.CmdCode == bolt.CmdCodeGoAway {
			log.DefaultLogger.Infof("got goaway frame: %v", req)
			client.goaway = true
		}

		if resp, ok = frame.(*BoltResponse); ok {
			break
		}

	}
	return string(resp.RawContent[:]), nil
}

func TestMosnForward(t *testing.T) {
	Scenario(t, "test mosn forward", func() {
		var m *mosn.MosnOperator
		server := &BoltServer{
			mode: NoGoAway,
		}
		Setup(func() {
			go startBoltServer(server)

			m = mosn.StartMosn(ConfigSimpleBoltExample)
			Verify(m, NotNil)
			time.Sleep(5 * time.Second) // wait mosn start
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
					log.DefaultLogger.Errorf("connect to server failed: %v", err)
				}
				defer conn.Close()
				client := &BoltClient{
					conn: conn,
				}
				var start time.Time
				start = time.Now()
				resp, err := boltRequest(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(server.connected, Equal, 1)
			}
		})
		TearDown(func() {
			stopBoltServer(server)
			m.Stop()
		})
	})
}

// server(mosn) send goaway frame when start graceful stop
func TestBoltGracefulStop(t *testing.T) {
	Scenario(t, "bolt graceful stop", func() {
		var m *mosn.MosnOperator
		server := &BoltServer{
			mode: NoGoAway,
		}
		Setup(func() {
			go startBoltServer(server)

			m = mosn.StartMosn(ConfigSimpleBoltExample)
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
				client := &BoltClient{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := boltRequest(client, tc.reqBody)
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
				resp, err = boltRequest(client, tc.reqBody)
				log.DefaultLogger.Infof("request cost %v", time.Since(start))
				Verify(err, Equal, nil)
				Verify(resp, Equal, tc.reqBody)
				Verify(client.goaway, Equal, true)
				Verify(server.connected, Equal, 1)
			}
		})
		TearDown(func() {
			stopBoltServer(server)
			m.Stop()
		})
	})
}

// client(mosn) got away frame before get response
func TestBoltServerGoAwayBeforeResponse(t *testing.T) {
	Scenario(t, "bolt server goaway", func() {
		var m *mosn.MosnOperator
		server := &BoltServer{
			mode: GoAwayBeforeResponse,
		}
		Setup(func() {
			go startBoltServer(server)

			m = mosn.StartMosn(ConfigSimpleBoltExample)
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
				client := &BoltClient{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := boltRequest(client, tc.reqBody)
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
				resp, err = boltRequest(client, tc.reqBody)
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
			stopBoltServer(server)
			m.Stop()
		})
	})
}

// client(mosn) got away frame after get response
func TestBoltServerGoAwayAfterResponse(t *testing.T) {
	Scenario(t, "bolt server goaway", func() {
		var m *mosn.MosnOperator
		server := &BoltServer{
			mode: GoAwayAfterResponse,
		}
		Setup(func() {
			go startBoltServer(server)

			m = mosn.StartMosn(ConfigSimpleBoltExample)
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
				client := &BoltClient{
					conn: conn,
				}
				// 1. simple
				var start time.Time
				start = time.Now()
				resp, err := boltRequest(client, tc.reqBody)
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
				resp, err = boltRequest(client, tc.reqBody)
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
			stopBoltServer(server)
			m.Stop()
		})
	})
}

const ConfigSimpleBoltExample = `{
    "servers": [
        {
            "default_log_path": "stdout",
            "default_log_level": "ERROR",
            "routers": [
                {
                    "router_config_name": "router_to_server",
                    "virtual_hosts": [
                        {
                            "name": "server_hosts",
                            "domains": [
                                "*"
                            ],
                            "routers": [
                                {
                                    "route": {
                                        "cluster_name": "server_cluster"
                                    }
                                }
                            ]
                        }
                    ]
                }
            ],
            "listeners": [
                {
                    "address": "127.0.0.1:2046",
                    "bind_port": true,
                    "filter_chains": [
                        {
                            "filters": [
                                {
                                    "type": "proxy",
                                    "config": {
                                        "downstream_protocol": "bolt",
                                        "router_config_name": "router_to_server",
                                        "extend_config": {
                                            "enable_bolt_goaway": true
                                        }
                                    }
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ],
    "cluster_manager": {
        "clusters": [
            {
                "name": "server_cluster",
                "type": "SIMPLE",
                "lb_type": "LB_RANDOM",
                "hosts": [
                    {
                        "address": "127.0.0.1:8080"
                    }
                ]
            }
        ]
    },
	"admin": {
		"address": {
			"socket_address": {
				"address": "0.0.0.0",
				"port_value": 34902
			}
		}
	}
}`
