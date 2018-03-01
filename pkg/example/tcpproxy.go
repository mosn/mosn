package main

import (
	"time"
	"net"
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
)

func main() {
	stopChan := make(chan bool)
	var srv server.Server

	go func() {
		// upstream
		l, _ := net.Listen("tcp", "127.0.0.1:8080")
		defer l.Close()

		for {
			select {
			case <-stopChan:
				break
			default:
				conn, _ := l.Accept()

				fmt.Println("realserver get connection..")
				fmt.Println()

				buf := make([]byte, 1024)
				conn.Read(buf)

				fmt.Printf(string(buf))
				conn.Write([]byte("Message received."))

				conn.Close()
			}
		}
	}()

	go func() {
		// mesh
		srv = server.NewServer(&proxy.TcpProxyFilterConfigFactory{
			Proxy: tcpProxyConfig(),
		})
		srv.AddListener(tcpProxyListener())
		srv.Start()
	}()

	go func() {
		// client
		remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2048")
		cc := network.NewClientConnection(nil, remoteAddr, stopChan)
		cc.AddConnectionCallbacks(&clientConnCallbacks{})
		cc.Connect()

		fmt.Println("write 'tst' to remote server")
		fmt.Println()

		buf := bytes.NewBufferString("tst")
		cc.Write(buf)
	}()

	select {
	case <-time.After(time.Second * 10):
		stopChan <- true
		srv.Close()
		fmt.Println("closing..")
	}
}

func tcpProxyListener() v2.ListenerConfig {
	return v2.ListenerConfig{
		Name:                 "tstListener",
		Addr:                 "127.0.0.1:2048",
		BindToPort:           true,
		ConnBufferLimitBytes: 1024 * 32,
	}
}

type clientConnCallbacks struct{}

func (ccc *clientConnCallbacks) OnEvent(event types.ConnectionEvent) {
	fmt.Printf("client connection event %s", string(event))
	fmt.Println()
}

func (ccc *clientConnCallbacks) OnAboveWriteBufferHighWatermark() {}

func (ccc *clientConnCallbacks) OnBelowWriteBufferLowWatermark() {}

func tcpProxyConfig() *v2.TcpProxy {
	tcpProxyConfig := &v2.TcpProxy{}
	tcpProxyConfig.Routes = append(tcpProxyConfig.Routes, &v2.TcpRoute{
		Cluster: "tstCluster",
	})

	return tcpProxyConfig
}
