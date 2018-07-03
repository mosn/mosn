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
	"fmt"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"

	"net"
	"time"
)

const (
	TestCluster    = "tstCluster"
	RealServerAddr = "127.0.0.1:8080"
)

func main2() {
	log.InitDefaultLogger("", log.DEBUG)

	stopChan := make(chan bool)
	upstreamReadyChan := make(chan bool)
	meshReadyChan := make(chan bool)

	go func() {
		// upstream
		l, _ := net.Listen("tcp", RealServerAddr)
		fmt.Println("listen on ")
		defer l.Close()

		for {
			select {
			case <-stopChan:
				break
			default:
				upstreamReadyChan <- true

				conn, _ := l.Accept()

				fmt.Printf("[REALSERVER]get connection %s..", conn.RemoteAddr())
				fmt.Println()

				buf := make([]byte, 4*1024)

				for {
					t := time.Now()
					conn.SetReadDeadline(t.Add(3 * time.Second))

					if bytesRead, err := conn.Read(buf); err != nil {

						if err, ok := err.(net.Error); ok && err.Timeout() {
							continue
						}

						fmt.Println("[REALSERVER]failed read buf")
						return
					} else {
						if bytesRead > 0 {
							fmt.Printf("[REALSERVER]get data '%s'", string(buf[:bytesRead]))
							fmt.Println()
							break
						}
					}
				}

				fmt.Printf("[REALSERVER]write back data 'world'")
				fmt.Println()

				conn.Write([]byte("world"))

				select {
				case <-stopChan:
					conn.Close()
				}
			}
		}
	}()

	go func() {
		select {
		case <-upstreamReadyChan:
			// mesh
			cmf := &clusterManagerFilter{}
			cm := cluster.NewClusterManager(nil, nil, nil, false, false)
			srv := server.NewServer(nil, cmf, cm)
			srv.AddListener(tcpListener(), &proxy.TcpProxyFilterConfigFactory{
				Proxy: tcpProxyConfig(),
			}, nil)
			cmf.cccb.UpdateClusterConfig(clusters())
			cmf.chcb.UpdateClusterHost(TestCluster, 0, hosts(""))

			meshReadyChan <- true

			srv.Start()

			select {
			case <-stopChan:
				srv.Close()
			}
		}
	}()

	go func() {
		select {
		case <-meshReadyChan:
			// client
			remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)
			cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
			cc.AddConnectionEventListener(&clientConnCallbacks{ //ADD  connection callback
				cc: cc,
			})
			cc.Connect(true)
			cc.SetReadDisable(false)
			cc.FilterManager().AddReadFilter(&clientConnReadFilter{})

			select {
			case <-stopChan:
				cc.Close(types.NoFlush, types.LocalClose)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}
}

type clientConnCallbacks struct {
	cc types.Connection
}

func (ccc *clientConnCallbacks) OnEvent(event types.ConnectionEvent) {
	fmt.Printf("[CLIENT]connection event %s", string(event))
	fmt.Println()

	switch event {
	case types.Connected:
		time.Sleep(3 * time.Second)

		fmt.Println("[CLIENT]write 'hello' to remote server")

		buf := buffer.NewIoBufferString("hello")
		ccc.cc.Write(buf)
	}
}

func (ccc *clientConnCallbacks) OnAboveWriteBufferHighWatermark() {}

func (ccc *clientConnCallbacks) OnBelowWriteBufferLowWatermark() {}

type clientConnReadFilter struct {
}

func (ccrf *clientConnReadFilter) OnData(buffer types.IoBuffer) types.FilterStatus {
	fmt.Printf("[CLIENT]receive data '%s'", buffer.String())
	fmt.Println()
	buffer.Reset()

	return types.Continue
}

func (ccrf *clientConnReadFilter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (ccrf *clientConnReadFilter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}