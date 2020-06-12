package connpool

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnpoolExample(t *testing.T) {
	// TODO
	// https://stackoverflow.com/questions/5134515/how-would-you-test-a-connection-pool
	conpool()
}

type userKeepAlive struct {}

func (kp *userKeepAlive) Stop() {}

func heartbeatReconnectTest() {}

func conpool() {
	server := tcpServer{
		addr : fmt.Sprint("127.0.0.1:10001"),
		//addr : fmt.Sprint("101.200.197.139:80"),
	}
	server.start()

	pool := NewConn(server.addr, 100000, func() KeepAlive2 { return nil }, nil, true)
	for i := 0; i < 100; i++ {
		//i := i
		go func() {
			/*
			for {
				buf := buffer.NewIoBufferString("bella")
				err := pool.Write(buf)
				if err != nil {
					fmt.Println("err is ", err.Error(), i, time.Now())
				} else {
					fmt.Println("write succ", i)
				}
			}

			 */
		}()
	}
	_ = pool

	go func() {
		for i:=0;i<1000;i++ {
			server.stop()
			time.Sleep(time.Millisecond*1000)
			server.start()
			time.Sleep(time.Second*5)
		}
	}()
	time.Sleep(time.Second * 105)
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
	atomic.StoreUint64(&s.started, 0)
}

func (s *tcpServer) closeAllConns() {
	for _, c := range s.conns {
		c.Close()
	}
}

func (s *tcpServer) start() {
	//if !atomic.CompareAndSwapUint64(&s.started, 0, 1) {
		// already started
	//	return
	//}

		var err error

		// listen on random port to avoid failure
		if s.addr == "" {
			// first init
			//s.addr = fmt.Sprintf("127.0.0.1:%v", rand.Intn(55532)+10000)
			s.addr = fmt.Sprintf("127.0.0.1:%v", 10001)
		}

		s.listener, err = net.Listen("tcp", s.addr)
		if err != nil {
			fmt.Println("listen failed", err)
		}

	go func() {
		for {
			c, e := s.listener.Accept()
			if e != nil {
				break
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
