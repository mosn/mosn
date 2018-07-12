/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work afor additional information regarding copyright ownership.
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

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	sf "gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"math/rand"
	"reflect"
	"sync/atomic"
)

//事例说明 (以下均支持可配置)

// 当前 server 数量为一个，监听端口 "127.0.0.1:8089"
// mosn监听端口: "127.0.0.1:2045"
// client 数目为三个，每 SendInterval 时间，发送一个请求
// client id = CloseClientNum，会在开启后5s后，关闭连接，并1s后重启

//使用说明 ()
// 根据以上配置，可以通过配置CloseClientNum来决定关闭哪个连接，在CloseClientNum > 2的情况下(例如这里是10)，可用于模拟正常case，即无连接关闭的情况
// 设置CloseClientNum = {0,1,2}等，可模拟由于client端链导致的问题
// CloseConnUpValue = 5  表示，close 连接的interval的最大值，实际值在此之间random

const (
	UpstreamAddr = "127.0.0.1:8089"
	//UpstreamAddr2      = "127.0.0.1:9099"

	MeshRPCServerAddr = "127.0.0.1:2045"
	TestClusterRPC    = "tstCluster"
	TestListenerRPC   = "tstListener"
)

// client相关配置
const (
	ClientNum        = 3
	CloseClientNum   = 1
	SendInterval     = 10 * time.Millisecond
	RestartTime      = 1 * time.Second
	ResponseTimeout  = 3000 * time.Millisecond
	CloseConnUpValue = 500
	Microsecond      = 1000
)

func main() {
	Run()
}

func Run() {
	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9099", nil)
	}()
	log.InitDefaultLogger("stderr", log.DEBUG)

	stopChan := make(chan bool)
	upstreamReadyChan := make(chan bool)
	meshReadyChan := make(chan bool)

	go func() {
		// upstream server
		l, _ := net.Listen("tcp", UpstreamAddr)
		defer l.Close()

		for {
			select {
			case <-stopChan:
				break
			case <-time.After(2 * time.Second):
				upstreamReadyChan <- true

				conn, _ := l.Accept()

				iobuf := buffer.NewIoBuffer(102400)

				for {
					t := time.Now()
					conn.SetReadDeadline(t.Add(3 * time.Second))

					buf := make([]byte, 10*1024)
					if bytesRead, err := conn.Read(buf); err != nil {

						if err, ok := err.(net.Error); ok && err.Timeout() {
							continue
						}

						fmt.Println("[REALSERVER]failed read buf")
						return
					} else {
						if bytesRead > 0 {
							iobuf.Write(buf[:bytesRead])

							//loop
							for iobuf.Len() > 1 {
								if read, cmd := codec.BoltV1.GetDecoder().Decode(nil, iobuf); cmd != nil {
									var resp *sofarpc.BoltResponseCommand

									if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
										fmt.Printf("%s Upstream Get Bolt Request, len = %d, request id = %d", time.Now().String(), read, req.ReqId)
										fmt.Println()
										resp = buildRespMag(req)
									}

									if err, iobufresp := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp); err == nil {
										//fmt.Printf("[REALSERVER] Write Bolt Response, len = %d, cmd = %+v'",len,result)
										//fmt.Println()

										respdata := iobufresp.Bytes()
										conn.Write(respdata)
									} else {
										fmt.Errorf("[REALSERVER]Write Bolt Response error = %+v \n", err)
									}
								} else {
									break
								}
							}
						}
					}
				}

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
			//  mesh
			cmf := &clusterManagerFilterRPC{}

			sh := &sf.HealthCheckFilterConfigFactory{
				FilterConfig: &v2.HealthCheckFilter{
					PassThrough: false,
					CacheTime:   3600,
				},
			}

			cm := cluster.NewClusterManager(nil, nil, nil, false, false)

			//RPC
			srv := server.NewServer(nil, cmf, cm)

			srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
				Proxy: genericProxyConfig(),
			}, []types.StreamFilterChainFactory{sh})

			cmf.cccb.UpdateClusterConfig(clustersrpc())
			cmf.chcb.UpdateClusterHost(TestClusterRPC, 0, rpchosts())

			meshReadyChan <- true
			fmt.Printf("%s Mosn Started!", time.Now().String())
			fmt.Println()

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
			var i uint32

			for i = 0; i < ClientNum; i++ {
				j := i
				time.Sleep(1 * time.Second)

				go func(tid uint32, stopC chan bool) {
					RunClientInstance(tid, stopC)
				}(j, stopChan)
			}
		}
	}()

	// exit only panic
	select {
	case <-time.After(time.Second * 1200):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}

}

// kill thread = 1 after run few seconds
func RunClientInstance(threadid uint32, stopChan chan bool) {
	fmt.Printf("new...threadid = %d \n", threadid)
	needReconnectFlag := false

	for {
		clientId := threadid

		allRequestMap := cmap.New()
		var streamIdCsounter uint32

		if needReconnectFlag {
			// 打印
			// 重置
			needReconnectFlag = false
			fmt.Printf("%s Client[%d] Reconnect to remote! ", time.Now().String(), clientId)
		}

		var closeClientTimer *time.Timer

		// 发起连接
		remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshRPCServerAddr)
		cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)

		if err := cc.Connect(true); err != nil {
			panic("connect to remote error")
		}

		codecClient := stream.NewCodecClient(nil, protocol.SofaRpc, cc, nil)

		//长连接上循环发送stream
		for {
			fmt.Println()

			// todo use increasing number
			id := atomic.AddUint32(&streamIdCsounter, 1)
			reqID := sofarpc.StreamIDConvert(id)

			requestMsg := RequestMsg{
				threadId: clientId,
				timer:    newTimer(RespTimeout, reqID, clientId),
				streamId: reqID,
			}
			requestMsg.allRequestMap = allRequestMap

			if CloseClientNum == clientId {
				go func() {
					closeClientTimer = time.NewTimer(time.Duration(rand.Intn(CloseConnUpValue) * Microsecond))

					select {
					case <-closeClientTimer.C:
						cc.Close(types.NoFlush, types.LocalClose)

						for _, key := range allRequestMap.Keys() {
							req, _ := allRequestMap.Get(key)

							if reqmsg, ok := req.(*RequestMsg); ok {
								reqmsg.timer.stop()
							}
							allRequestMap.Remove(key)
						}

						fmt.Printf("%s Client[%d] close connection!", time.Now().String(), clientId)
						needReconnectFlag = true
					}
				}()
			}

			fmt.Printf("%s Client[%d] Send Request, RequestID = %s ", time.Now().String(), clientId, reqID)
			fmt.Println()

			allRequestMap.Set(reqID, &requestMsg)
			requestEncoder := codecClient.NewStream(reqID, &requestMsg)
			reqHeaders := buildRequestMsg(id)
			requestEncoder.AppendHeaders(reqHeaders, true)

			requestMsg.timer.start(ResponseTimeout)

			// 跳出内部收发循环，发起新的连接
			if needReconnectFlag {
				break
			}

			time.Sleep(SendInterval)
		}

		// 1s后拉起
		time.Sleep(RestartTime)
	}
}

type RequestMsg struct {
	streamId      string
	threadId      uint32
	timer         *timer
	allRequestMap cmap.ConcurrentMap
}

func (s *RequestMsg) OnReceiveHeaders(headers map[string]string, endStream bool) {
	//bolt
	s.timer.stop()
	s.allRequestMap.Remove(s.streamId)

	if statusStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]; ok {
		if sofarpc.ConvertPropertyValue(statusStr, reflect.Int16).(int16) == 0 {
			fmt.Printf("%s Client[%d] Get Success Response, RequestID = %s",
				time.Now().String(), s.threadId, s.streamId)
			fmt.Println()
		} else {
			panic("client get response failure")
		}
	} else {
		panic("client get response failure")
	}

	return
}

func (s *RequestMsg) OnReceiveData(data types.IoBuffer, endStream bool) {
}

func (s *RequestMsg) OnReceiveTrailers(trailers map[string]string) {
}

func (s *RequestMsg) OnDecodeError(err error, headers map[string]string) {
}

// build request msg
// 22固定字节头部 + headermap长度
func buildRequestMsg(requestId uint32) *sofarpc.BoltRequestCommand {

	request := &sofarpc.BoltRequestCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqId:    requestId,
		CodecPro: sofarpc.HESSIAN_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	headers := map[string]string{"service": "testSofa"} // used for sofa routing

	if headerBytes, err := serialize.Instance.Serialize(headers); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBytes
		request.HeaderLen = int16(len(headerBytes))
	}

	return request
}

// response报文对于request报文来说，基本内容不变
func buildRespMag(req *sofarpc.BoltRequestCommand) *sofarpc.BoltResponseCommand {
	return &sofarpc.BoltResponseCommand{
		Protocol:       req.Protocol,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        req.Version,
		ReqId:          req.ReqId,
		CodecPro:       req.CodecPro, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		HeaderLen:      req.HeaderLen,
		HeaderMap:      req.HeaderMap,
	}
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

// proxy config
func genericProxyConfig() *v2.Proxy {
	proxyConfig := &v2.Proxy{
		DownstreamProtocol: string(protocol.SofaRpc),
		UpstreamProtocol:   string(protocol.SofaRpc),
	}

	header := v2.HeaderMatcher{
		Name:  "service",
		Value: ".*",
	}

	routerV2 := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header},
		},

		Route: v2.RouteAction{
			ClusterName: TestClusterRPC,
		},
	}

	proxyConfig.VirtualHosts = append(proxyConfig.VirtualHosts, &v2.VirtualHost{
		Name:    "testSofaRoute",
		Domains: []string{"*"},
		Routers: []v2.Router{routerV2},
	})

	return proxyConfig
}

// mesh listener
func rpcProxyListener() *v2.ListenerConfig {
	addr, _ := net.ResolveTCPAddr("tcp", MeshRPCServerAddr)

	return &v2.ListenerConfig{
		Name:                    TestListenerRPC,
		Addr:                    addr,
		BindToPort:              true,
		PerConnBufferLimitBytes: 1024 * 32,
		LogPath:                 "stderr",
		LogLevel:                uint8(log.DEBUG),
	}
}

// mesh host
func rpchosts() []v2.Host {
	var hosts []v2.Host

	hosts = append(hosts, v2.Host{
		Address: UpstreamAddr,
		Weight:  100,
	})

	return hosts
}

func RespTimeout(reqid string, clientid uint32) {
	fmt.Printf("panic request id = %s, client id = %d", reqid, clientid)
	panic("client get response timeout")
}

// timer
type timer struct {
	clientID    uint32
	reqID       string
	callback    func(string, uint32)
	interval    time.Duration
	innerTimer  *time.Timer
	startedFlag bool
}

func newTimer(callback func(reqID string, clientID uint32), reqID string, clientID uint32) *timer {
	return &timer{
		callback: callback,
		reqID:    reqID,
		clientID: clientID,
	}
}

func (t *timer) start(interval time.Duration) {
	t.innerTimer = time.NewTimer(interval)
	t.startedFlag = true
	go func() {
		select {
		case <-t.innerTimer.C:
			t.callback(t.reqID, t.clientID)
		}
	}()
}

func (t *timer) stop() {
	if !t.startedFlag {
		return
	}
	
	fmt.Println("stop timer")
	t.innerTimer.Stop()
}
