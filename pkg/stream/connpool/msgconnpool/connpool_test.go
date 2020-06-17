package msgconnpool

import (
	"fmt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockKeepalive struct {
	// should record every heart beat frame callback
	// should handle the heart beat frame response timeout
	heartbeatFailCallback map[int]func()

	keepaliveFrameTimeout time.Duration

	heartbeatTriggerCount int
}

// GetKeepAliveData get the heartbeat frame
//  according to the app level protocol
func (kp *mockKeepalive) GetKeepAliveData(failCallback func()) []byte {
	heartbeatFrameID := 1
	kp.heartbeatFailCallback[heartbeatFrameID] = failCallback

	time.AfterFunc(kp.keepaliveFrameTimeout, func(){
		kp.heartbeatFrameTimeout(heartbeatFrameID)
	})

	kp.heartbeatTriggerCount++

	return []byte("yes this is the heartbeat data" + fmt.Sprint(heartbeatFrameID))
}

func (kp *mockKeepalive) heartbeatFrameTimeout(frameID int) {
	callback := kp.heartbeatFailCallback[frameID]

	if callback != nil {
		println("heartbeat frame timeout, callback")
		callback()
		delete(kp.heartbeatFailCallback, frameID)
	}
}

func (kp *mockKeepalive) Stop() {
	kp.heartbeatFailCallback = map[int]func(){}
}

func TestHeartBeatFail(t *testing.T) {
	// setup
	buffer.ConnReadTimeout = time.Second * 2
	// tear down
	defer func() {
		buffer.ConnReadTimeout = types.DefaultConnReadTimeout
	}()

	server := tcpServer{
		addr: fmt.Sprint("127.0.0.1:10001"),
	}

	mkp := &mockKeepalive{
		heartbeatFailCallback : make(map[int]func()),
		keepaliveFrameTimeout : buffer.ConnReadTimeout - time.Second,
	}
	pool := NewConn(server.addr, -1, func() KeepAlive {
		return mkp
	}, nil, true)
	server.startServer(false)
	defer server.stop()
	// default read time out is 15 seconds
	time.Sleep(buffer.ConnReadTimeout + time.Second * 5)

	assert.Equal(t, pool.Available(), true)
	assert.LessOrEqual(t, 0, mkp.heartbeatTriggerCount)
}

func TestReconnectTimesLimit(t *testing.T) {
	invalidAddr := "127.0.0.1:12345"
	tryTimes := 2
	pool := NewConn(invalidAddr, tryTimes, func() KeepAlive { return nil }, nil, true)
	time.Sleep(time.Second * 10)
	poolReal := pool.(*connpool)
	fmt.Println("retry for", poolReal.connTryTimes, "times")

	assert.LessOrEqual(t, poolReal.connTryTimes, tryTimes)
}

func TestAutoReconnectAfterRemoteClose(t *testing.T) {
	server := tcpServer{
		addr: fmt.Sprint("127.0.0.1:10001"),
	}

	pool := NewConn(server.addr, 100000, func() KeepAlive { return nil }, nil, true)
	server.startServer(true)
	defer server.stop()

	time.Sleep(time.Second * 5)
	// the connection should be connected
	assert.Equal(t, pool.Available(), true)
}


type tcpServer struct {
	listener net.Listener
	addr     string
	started  uint64

	conns []net.Conn
}

func (s *tcpServer) stop() {
	s.listener.Close()
	for _, c := range s.conns {
		c.Close()
	}

	s.conns = nil
}

func (s *tcpServer) startServer(rejectFirstConn bool) {
	var err error

	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		fmt.Println("listen failed", err)
	}

	go func() {
		for i := 0; ; i++ {
			c, e := s.listener.Accept()
			if e != nil {
				break
			}

			if i == 0 && rejectFirstConn {
				c.Close()
				continue
			}

			s.conns = append(s.conns, c)
			go func() {
				var buf = make([]byte, 1024)
				for {
					n, e := c.Read(buf)
					if e != nil {
						break
					}

					buf = buf[:n]
				}
			}()
		}
	}()
}
