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

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/stream/http2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"golang.org/x/net/http2"
	"encoding/hex"
)

const (
	RealServerAddr  = "127.0.0.1:8088"
	RealServerAddr2 = "127.0.0.1:9099"

	MeshServerAddr = "127.0.0.1:2045"
	TestCluster    = "tstCluster"
	TestListener   = "tstListener"
)

var boltV1ReqBytes []byte

func main() {
	boltV1ReqBytes, _ = hex.DecodeString("0101000101000000010100001388002c002e000005b5636f6d2e616c697061792e736f66612e7270632e636f72652e726571756573742e536f66615265717565737400000007736572766963650000001f636f6d2e616c697061792e746573742e54657374536572766963653a312e304fbc636f6d2e616c697061792e736f66612e7270632e636f72652e726571756573742e536f666152657175657374950d7461726765744170704e616d650a6d6574686f644e616d651774617267657453657276696365556e697175654e616d650c7265717565737450726f70730d6d6574686f64417267536967736f904e076563686f5374721f636f6d2e616c697061792e746573742e54657374536572766963653a312e304d03617070037878780870726f746f636f6c04626f6c74117270635f74726163655f636f6e746578744d09736f66615270634964013007456c61737469634e0b73797350656e4174747273000d736f666143616c6c657249646300097a70726f78795549444e107a70726f78795461726765745a6f6e654e0c736f666143616c6c65724970000b736f6661547261636549641d30613066653865663135323431343435383331373231303031393836300c736f666150656e4174747273000e736f666143616c6c65725a6f6e654e097a70726f78795669704e0d736f666143616c6c6572417070037878787a7a567400075b737472696e676e01106a6176612e6c616e672e537472696e677a53040031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334")
	
	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9099", nil)
	}()

	log.InitDefaultLogger("", log.DEBUG)

	stopChan := make(chan bool)
	meshReadyChan := make(chan bool)

	go func() {
		// upstream
		server := &http.Server{
			Addr:         ":8080",
			Handler:      &serverHandler{},
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		s2 := &http2.Server{
			IdleTimeout: 1 * time.Minute,
		}

		http2.ConfigureServer(server, s2)
		l, _ := net.Listen("tcp", RealServerAddr)
		defer l.Close()

		for {
			rwc, err := l.Accept()
			if err != nil {
				fmt.Println("accept err:", err)
				continue
			}
			go s2.ServeConn(rwc, &http2.ServeConnOpts{BaseConfig: server})
		}
	}()

	select {
	case <-time.After(2 * time.Second):
	}

	go func() {
		//  mesh
		cmf := &clusterManagerFilterRPC{}

		sf := &faultinject.FaultInjectFilterConfigFactory{
			FaultInject: &v2.FaultInject{
				DelayPercent:  100,
				DelayDuration: 2000,
			},
		}

		sh := &sofarpc.HealthCheckFilterConfigFactory{
			FilterConfig: &v2.HealthCheckFilter{
				PassThrough: false,
				CacheTime:   3600,
			},
		}

		cm := cluster.NewClusterManager(nil, nil, nil, false, false)

		//RPC
		srv := server.NewServer(&server.Config{
			LogPath:  "stderr",
			LogLevel: log.DEBUG,
		}, cmf, cm)

		srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
			Proxy: genericProxyConfig(),
		}, []types.StreamFilterChainFactory{sf, sh})
		cmf.cccb.UpdateClusterConfig(clustersrpc())
		cmf.chcb.UpdateClusterHost(TestCluster, 0, rpchosts())

		meshReadyChan <- true

		srv.Start() //开启连接

		select {
		case <-stopChan:
			srv.Close()
		}
	}()

	go func() {
		select {
		case <-meshReadyChan:
			// client
			remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)
			cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
			cc.AddConnectionEventListener(&rpclientConnCallbacks{ //ADD  connection callback
				cc: cc,
			})
			cc.Connect(true)
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

type serverHandler struct{}

func (sh *serverHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ShowRequestInfoHandler(w, req)
}

func ShowRequestInfoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s", r.URL)
	fmt.Println()

	w.Header().Set("Content-Type", "text/plain")

	for k, _ := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "RequestURI: %q\n", r.RequestURI)
	fmt.Fprintf(w, "URL: %#v\n", r.URL)
	fmt.Fprintf(w, "Body.ContentLength: %d (-1 means unknown)\n", r.ContentLength)
	fmt.Fprintf(w, "Close: %v (relevant for HTTP/1 only)\n", r.Close)
	fmt.Fprintf(w, "TLS: %#v\n", r.TLS)
	fmt.Fprintf(w, "\nHeaders:\n")

	r.Header.Write(w)
}

func genericProxyConfig() *v2.Proxy {
	proxyConfig := &v2.Proxy{
		DownstreamProtocol: string(protocol.SofaRpc),
		UpstreamProtocol:   string(protocol.Http2),
	}

	header := v2.HeaderMatcher{
		Name:  "service",
		Value: ".*",
	}

	var envoyvalue = map[string]interface{}{"stage": "pre-release", "version": "1.1", "label": "gray"}

	var value = map[string]interface{}{"mosn.lb": envoyvalue}

	routerV2 := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header},
		},

		Route: v2.RouteAction{
			ClusterName: TestCluster,
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
	addr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)

	return &v2.ListenerConfig{
		Name:                    TestListener,
		Addr:                    addr,
		BindToPort:              true,
		PerConnBufferLimitBytes: 1024 * 32,
		LogPath:                 "stderr",
		LogLevel:                uint8(log.DEBUG),
		AccessLogs:              []v2.AccessLog{{Path: "stderr"}},
	}
}

func rpchosts() []v2.Host {
	var hosts []v2.Host

	hosts = append(hosts, v2.Host{
		Address: RealServerAddr,
		Weight:  100,
		MetaData: map[string]interface{}{
			"stage":   "pre-release",
			"version": "1.1",
			"label":   "gray",
		},
	})

	hosts = append(hosts, v2.Host{
		Address: RealServerAddr2,
		Weight:  100,
		MetaData: map[string]interface{}{
			"stage":   "pre-release",
			"version": "1.2",
			"label":   "blue",
		},
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
	var lbsubsetconfig = v2.LBSubsetConfig{

		FallBackPolicy: 2,
		DefaultSubset: map[string]string{
			"stage":   "pre-release",
			"version": "1.1",
			"label":   "gray",
		},
		SubsetSelectors: [][]string{{"stage", "type"},
			{"stage", "label", "version"},
			{"version"}},
	}

	configs = append(configs, v2.Cluster{
		Name:                 TestCluster,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 32 * 1024,
		CirBreThresholds:     v2.CircuitBreakers{},
		LBSubSetConfig:       lbsubsetconfig,
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
	fmt.Println()
	fmt.Println("[CLIENT]Receive data:")
	fmt.Printf("%s", buffer.String())
	buffer.Reset()

	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}
