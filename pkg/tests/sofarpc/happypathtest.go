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
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"

	"encoding/hex"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

const (
	RealRPCServerAddr = "127.0.0.1:8089"
	MeshRPCServerAddr = "127.0.0.1:2045"
	TestClusterRPC    = "tstCluster"
	TestListenerRPC   = "tstListener"
)

var boltV1ReqBytes, boltV1ResBytes []byte

func main() {

	boltV1ReqBytes, _ = hex.DecodeString("0101000101000000010100001388002c002e000005b5636f6d2e616c697061792e736f66612e7270632e636f72652e726571756573742e536f66615265717565737400000007736572766963650000001f636f6d2e616c697061792e746573742e54657374536572766963653a312e304fbc636f6d2e616c697061792e736f66612e7270632e636f72652e726571756573742e536f666152657175657374950d7461726765744170704e616d650a6d6574686f644e616d651774617267657453657276696365556e697175654e616d650c7265717565737450726f70730d6d6574686f64417267536967736f904e076563686f5374721f636f6d2e616c697061792e746573742e54657374536572766963653a312e304d03617070037878780870726f746f636f6c04626f6c74117270635f74726163655f636f6e746578744d09736f66615270634964013007456c61737469634e0b73797350656e4174747273000d736f666143616c6c657249646300097a70726f78795549444e107a70726f78795461726765745a6f6e654e0c736f666143616c6c65724970000b736f6661547261636549641d30613066653865663135323431343435383331373231303031393836300c736f666150656e4174747273000e736f666143616c6c65725a6f6e654e097a70726f78795669704e0d736f666143616c6c6572417070037878787a7a567400075b737472696e676e01106a6176612e6c616e672e537472696e677a53040031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334")
	boltV1ResBytes, _ = hex.DecodeString("010000020100000001010000002e000000000464636f6d2e616c697061792e736f66612e7270632e636f72652e726573706f6e73652e536f6661526573706f6e73654fbe636f6d2e616c697061792e736f66612e7270632e636f72652e726573706f6e73652e536f6661526573706f6e7365940769734572726f72086572726f724d73670b617070526573706f6e73650d726573706f6e736550726f70736f90464e530400313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233344e")
	fmt.Println(len(boltV1ReqBytes), len(boltV1ResBytes))
	Run()
}

func genericProxyConfig() *v2.Proxy {
	proxyConfig := &v2.Proxy{
		DownstreamProtocol: string(protocol.SofaRpc),
		UpstreamProtocol:   string(protocol.SofaRpc),
	}

	header := v2.HeaderMatcher{
		Name:  "service",
		Value: "com.alipay.test.TestService:1.0",
	}

	var envoyvalue = map[string]interface{}{"stage": "pre-release", "version": "1.1", "label": "gray"}

	var value = map[string]interface{}{"mosn.lb": envoyvalue}

	routerV2 := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header},
		},

		Route: v2.RouteAction{
			ClusterName: TestClusterRPC,
			MetadataMatch: v2.Metadata{
				"filter_metadata": value,
			},
		},
	}

	proxyConfig.VirtualHosts = append(proxyConfig.VirtualHosts, &v2.VirtualHost{
		Name:    "testSofaRoute",
		Domains: []string{"*"},
		Routers: []v2.Router{routerV2},
	})

	return proxyConfig
}

func rpcProxyListener() *v2.ListenerConfig {
	addr, _ := net.ResolveTCPAddr("tcp", MeshRPCServerAddr)

	return &v2.ListenerConfig{
		Name:                    TestListenerRPC,
		Addr:                    addr,
		BindToPort:              true,
		PerConnBufferLimitBytes: 1024 * 32,
		LogPath:                 "stdout",
		LogLevel:                uint8(log.DEBUG),
	}
}

func rpchosts() []v2.Host {
	var hosts []v2.Host

	hosts = append(hosts, v2.Host{
		Address: RealRPCServerAddr,
		Weight:  100,
	})

	return hosts
}

type clusterManagerFilterRPC struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilterRPC) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func clustersrpc() []v2.Cluster {
	var configs []v2.Cluster
	configs = append(configs, v2.Cluster{
		Name:                 TestClusterRPC,
		ClusterType:          v2.SIMPLE_CLUSTER,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 32 * 1024,
		CirBreThresholds:     v2.CircuitBreakers{},
	})

	return configs
}

type rpclientConnCallbacks struct {
	cc types.Connection
}

func (ccc *rpclientConnCallbacks) OnEvent(event types.ConnectionEvent) {
	fmt.Printf("[CLIENT]connection event %s", string(event))
	fmt.Println()

	switch event {
	case types.Connected:
		time.Sleep(3 * time.Second)

		fmt.Println("[CLIENT]write 'bolt test msg' to remote server")

		boltV1PostData := buffer.NewIoBufferBytes(boltV1ReqBytes)

		ccc.cc.Write(boltV1PostData)
	}
}

type rpcclientConnReadFilter struct {
}

func (ccrf *rpcclientConnReadFilter) OnData(buffer types.IoBuffer) types.FilterStatus {
	//s := buffer.String()
	fmt.Printf("[Client Receive]: %s \n","Bolt Response")
	
	buffer.Reset()

	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

func Run() {
	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9099", nil)
	}()

	log.InitDefaultLogger("stdout", log.DEBUG)

	stopChan := make(chan bool)
	upstreamReadyChan := make(chan bool)
	meshReadyChan := make(chan bool)

	go func() {
		//upstream
		l, _ := net.Listen("tcp", RealRPCServerAddr)
		
		defer l.Close()
		
		for {
			select {
			case <-stopChan:
				break
			case <-time.After(2 * time.Second):
				upstreamReadyChan <- true
		
				conn, _ := l.Accept()
		
				fmt.Printf("[REALSERVER]get connection %s..\n", conn.RemoteAddr())
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
							fmt.Printf("[REALSERVER] Get Bolt Request'")
							break
						}
					}
				}
		
				fmt.Printf("[REALSERVER]write back data 'Got Bolt Msg'\n")
		
				conn.Write(boltV1ResBytes)
		
				select {
				case <-stopChan:
					conn.Close()
				}
			}
		}
		upstreamReadyChan <- true
	}()

	go func() {
		select {
		case <-upstreamReadyChan:
			//  mesh
			cmf := &clusterManagerFilterRPC{}

			cm := cluster.NewClusterManager(nil, nil, nil, false, false)

			//RPC
			srv := server.NewServer(nil, cmf, cm)

			srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
				Proxy: genericProxyConfig(),
			}, nil)
			cmf.cccb.UpdateClusterConfig(clustersrpc())
			cmf.chcb.UpdateClusterHost(TestClusterRPC, 0, rpchosts())

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
			remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshRPCServerAddr)
			cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
			cc.AddConnectionEventListener(&rpclientConnCallbacks{ //ADD  connection callback
				cc: cc,
			})
			cc.Connect(true)
			cc.SetReadDisable(false)
			cc.FilterManager().AddReadFilter(&rpcclientConnReadFilter{})

			select {
			case <-stopChan:
				cc.Close(types.NoFlush, types.LocalClose)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 120):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}

}
