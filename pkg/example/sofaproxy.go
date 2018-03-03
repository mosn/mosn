package main


import (
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"


	//"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	//"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"time"
	"net"
)

const (
	RealRPCServerAddr = "127.0.0.1:8088"
	MeshRPCServerAddr = "127.0.0.1:2044"
)


func main() {

	//test_codec()
	//initilize codec engine. TODO: config driven
	codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
		sofarpc.PROTOCOL_CODE_V1:sofarpc.BoltV1,
		sofarpc.PROTOCOL_CODE_V2:sofarpc.BoltV2,
		sofarpc.PROTOCOL_CODE:sofarpc.Tr,

	})

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
			//  mesh
			cmf := &clusterManagerFilter{}

			//RPC
			srv := server.NewServer(&proxy.RpcProxyFilterConfigFactory{
				Proxy: rpcProxyConfig(),
			}, cmf)


			srv.AddListener(rpcProxyListener())
			cmf.cccb.UpdateClusterConfig(clusters())
			cmf.chcb.UpdateClusterHost(TestCluster, 0, rpchosts())

			meshReadyChan <- true

			srv.Start()

			select {
			case <-stopChan:
				srv.Close()
			}
		}
	}()






}
func rpcProxyConfig() *v2.RpcProxy {
	rpcProxyConfig := &v2.RpcProxy{}
	rpcProxyConfig.Routes = append(tcpProxyConfig.Routes, &v2.RpcRoute{
		Cluster: TestCluster,
	})

	return rpcProxyConfig
}

func rpcProxyListener() v2.ListenerConfig {
	return v2.ListenerConfig{
		Name:                 TestListener,
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

