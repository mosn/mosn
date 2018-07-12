package tests
//
//import (
//	"crypto/tls"
//	"fmt"
//	"io/ioutil"
//	"net"
//	"net/http"
//	"testing"
//	"time"
//
//	"github.com/alipay/sofamosn/pkg/mosn"
//	"github.com/alipay/sofamosn/pkg/protocol"
//	"golang.org/x/net/http2"
//)
//
//func TestHttp2(t *testing.T) {
//	meshAddr := "127.0.0.1:2045"
//	http2Addr := "127.0.0.1:8080"
//	server := NewUpstreamHttp2(t, http2Addr)
//	server.GoServe()
//	defer server.Close()
//	mesh_config := CreateSimpleMeshConfig(meshAddr, []string{http2Addr}, protocol.Http2, protocol.Http2)
//	go mosn.Start(mesh_config, "", "")
//	time.Sleep(5 * time.Second) //wait mesh and server start
//	//Client Run
//	tr := &http2.Transport{
//		AllowHTTP: true,
//		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
//			return net.Dial(netw, addr)
//		},
//	}
//
//	httpClient := http.Client{Transport: tr}
//	for i := 0; i < 20; i++ {
//		requestId := fmt.Sprintf("%d", i)
//		request, err := http.NewRequest("GET", fmt.Sprintf("http://%s", meshAddr), nil)
//		if err != nil {
//			t.Fatalf("create request error:%v\n", err)
//		}
//		request.Header.Add("service", "testhttp2")
//		request.Header.Add("Requestid", requestId)
//		resp, err := httpClient.Do(request)
//		if err != nil {
//			t.Errorf("request %s response error: %v\n", requestId, err)
//			continue
//		}
//		defer resp.Body.Close()
//		body, err := ioutil.ReadAll(resp.Body)
//		if err != nil {
//			t.Errorf("request %s read body error: %v\n", requestId, err)
//			continue
//		}
//		t.Logf("request %s get data: %s\n", requestId, body)
//	}
//
//}
