package log
//
//import (
//	"testing"
//	"github.com/alipay/sofamosn/pkg/network"
//	"net"
//	"time"
//	"fmt"
//)
//
//func BenchmarkAccessLog(b *testing.B) {
//	InitDefaultLogger("", INFO)
//	// ~ replace the path if needed
//	accessLog, err := NewAccessLog("/tmp/mosn_bench/benchmark_access.log", nil, "")
//
//	if err != nil {
//		fmt.Errorf(err.Error())
//	}
//	reqHeaders := map[string]string{
//		"service": "test",
//	}
//
//	respHeaders := map[string]string{
//		"Server": "MOSN",
//	}
//
//	requestInfo := network.NewRequestInfoWithPort("Http1")
//	requestInfo.SetRequestReceivedDuration(time.Now())
//	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
//	requestInfo.SetBytesSent(2048)
//	requestInfo.SetBytesReceived(2048)
//
//	requestInfo.SetResponseFlag(0)
//	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 23456, ""})
//	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 12200, ""})
//	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{[]byte("127.0.0.2"), 53242, ""})
//	requestInfo.OnUpstreamHostSelected(nil)
//
//	for n := 0; n < b.N; n++ {
//		accessLog.Log(reqHeaders, respHeaders, requestInfo)
//	}
//}