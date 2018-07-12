package main

import (
	"fmt"
	"net"
	"time"

	"github.com/alipay/sofamosn/pkg/log"
	"github.com/alipay/sofamosn/pkg/network"
	"github.com/alipay/sofamosn/pkg/network/buffer"
	"github.com/alipay/sofamosn/pkg/types"
)

func main() {
	//MeshServerAddr := "127.0.0.1:2046"
	//MeshServerAddr := "11.166.22.163:12200"   // c++ 的测试后台
	MeshServerAddr := "127.0.0.1:8080"   // 直接发往server
	
	stopChan := make(chan bool)
	log.InitDefaultLogger("", log.DEBUG)
	remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	cc.AddConnectionEventListener(&rpclientConnCallbacks{
		cc: cc,
	})
	cc.Connect(true)
	cc.FilterManager().AddReadFilter(&rpcclientConnReadFilter{})
	//wait response
	<-stopChan
	//	<-time.After(10 * time.Second)
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

		fmt.Println("[CLIENT]write 'bolt message' to remote server")

		boltV1PostData := buffer.NewIoBufferBytes(boltV1ReqBytes)

		ccc.cc.Write(boltV1PostData)
	}
}

func (ccc *rpclientConnCallbacks) OnAboveWriteBufferHighWatermark() {}

func (ccc *rpclientConnCallbacks) OnBelowWriteBufferLowWatermark() {}

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
