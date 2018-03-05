package main


import (
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"


	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"time"
	"net"
)

const (
	RealRPCServerAddr = "127.0.0.1:8088"
	MeshRPCServerAddr = "127.0.0.1:2044"
	TestClusterRPC    = "tstCluster"
	TestListenerRPC   = "tstListener"
)


func main() {

	//test_codec()
	//initilize codec engine. TODO: config driven
	//codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
	//	sofarpc.PROTOCOL_CODE_V1:sofarpc.BoltV1,
	//	sofarpc.PROTOCOL_CODE_V2:sofarpc.BoltV2,
	//	sofarpc.PROTOCOL_CODE:sofarpc.Tr,
	//
	//})

	stopChan := make(chan bool)
	upstreamReadyChan := make(chan bool)
	meshReadyChan := make(chan bool)


	go func() {
		// upstream
		l, _ := net.Listen("tcp", RealRPCServerAddr)

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

				fmt.Printf("[REALSERVER]write back data 'Got Bolt Msg'")
				fmt.Println()

				conn.Write([]byte("Got Bolt"))

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

			//RPC
			srv := server.NewServer(&proxy.RpcProxyFilterConfigFactory{
				Proxy: rpcProxyConfig(),
			}, cmf)

	//		boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1"))
			//codecImpl.Decode(nil,boltV1PostData,nil)

			//

			srv.AddListener(rpcProxyListener())
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
			cc := network.NewClientConnection(nil, remoteAddr, stopChan)
			cc.AddConnectionCallbacks(&rpclientConnCallbacks{      //ADD  connection callback
				cc: cc,
			})
			cc.Connect()
			cc.SetReadDisable(false)
			cc.FilterManager().AddReadFilter(&rpcclientConnReadFilter{})

			select {
			case <-stopChan:
				cc.Close(types.NoFlush)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}





}
func rpcProxyConfig() *v2.RpcProxy {
	rpcProxyConfig := &v2.RpcProxy{}
	rpcProxyConfig.Routes = append(rpcProxyConfig.Routes, &v2.RpcRoute{
		Cluster: TestClusterRPC,
	})

	return rpcProxyConfig
}

func rpcProxyListener() v2.ListenerConfig {
	return v2.ListenerConfig{
		Name:                 TestListenerRPC,
		Addr:                 MeshRPCServerAddr,
		BindToPort:           true,
		ConnBufferLimitBytes: 1024 * 32,
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


//
func test_codec(){

	//initilize codec engine. TODO: config driven
	codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
		sofarpc.PROTOCOL_CODE_V1:sofarpc.BoltV1,
		sofarpc.PROTOCOL_CODE_V2:sofarpc.BoltV2,
		sofarpc.PROTOCOL_CODE:sofarpc.Tr,

	})

	//plug-in tr codec
	//codecImpl.PutProtocol(sofarpc.PROTOCOL_CODE, sofarpc.Tr)


	trPostData := bytes.NewBuffer([]byte("\x0d\x00TaobaoRemoting"))
	boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1"))
	boltV2PostData := bytes.NewBuffer([]byte("\x02\x00BoltV2"))

	//test tr decode branch
	fmt.Println("-----------> tr test begin")
	codecImpl.Decode(nil, trPostData, nil)
	fmt.Println("<----------- tr test end\n")

	//test boltv1 decode branch
	fmt.Println("-----------> boltv1 test begin")
	codecImpl.Decode(nil, boltV1PostData, nil)
	fmt.Println("<----------- boltv1 test end\n")

	//test boltv2 decode branch
	fmt.Println("-----------> boltv2 test begin")
	codecImpl.Decode(nil, boltV2PostData, nil)
	fmt.Println("<----------- boltv2 test end\n")
}

type clusterManagerFilterRPC struct {
	cccb server.ClusterConfigFactoryCb
	chcb server.ClusterHostFactoryCb
}


func (cmf *clusterManagerFilterRPC) OnCreated(cccb server.ClusterConfigFactoryCb, chcb server.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}


func clustersrpc() []v2.Cluster {
	var configs []v2.Cluster
	configs = append(configs, v2.Cluster{
		Name:                 TestClusterRPC,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 16 * 1026,
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

		//buf := bytes.NewBufferString("hello")
		boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1test"))
		//ccc.cc.Write(buf)
		ccc.cc.Write(boltV1PostData)
	}
}

func (ccc *rpclientConnCallbacks) OnAboveWriteBufferHighWatermark() {}

func (ccc *rpclientConnCallbacks) OnBelowWriteBufferLowWatermark() {}


type rpcclientConnReadFilter struct {
}

func (ccrf *rpcclientConnReadFilter) OnData(buffer *bytes.Buffer) types.FilterStatus {
	fmt.Printf("[CLIENT]receive data '%s'", buffer.String())
	fmt.Println()
	buffer.Reset()

	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}