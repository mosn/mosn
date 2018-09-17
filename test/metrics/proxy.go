package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"time"

	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/stats"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

func main() {
	p := flag.String("p", "sofarpc", "-p=http1/http2/sofarpc")
	flag.Parse()
	var server Server
	var client Client
	var proto types.Protocol
	meshAddr := "127.0.0.1:2045"
	serverAddr := "127.0.0.1:8080"
	switch *p {
	case "http1":
		server = NewHTTP1Server(serverAddr)
		client = NewHTTP1Client(meshAddr)
		proto = protocol.HTTP1
	case "http2":
		server = NewHTTP2Server(serverAddr)
		client = NewHTTP2Client(meshAddr)
		proto = protocol.HTTP2
	case "sofarpc":
		server = NewRPCServer(serverAddr)
		client = NewRPCClient(meshAddr)
		proto = protocol.SofaRPC
	default:
		return
	}
	cfg := util.CreateProxyMesh(meshAddr, []string{serverAddr}, proto)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go server.Start()
	// Proxy API
	proxy := &Proxy{client}
	http.HandleFunc("/send", proxy.SendRequest)
	http.HandleFunc("/destroy", proxy.DestroyConn)
	http.HandleFunc("/stats", proxy.Stats)
	http.ListenAndServe("127.0.0.1:8081", nil)
}

type Proxy struct {
	client Client
}

func (p *Proxy) SendRequest(w http.ResponseWriter, r *http.Request) {
	ch := p.client.Send()
	select {
	case err := <-ch:
		if err != nil {
			w.Write([]byte(err.Error() + "\n"))
		} else {
			w.Write([]byte("success\n"))
		}
	case <-time.After(10 * time.Second):
		w.WriteHeader(503)
		w.Write([]byte("timeout\n"))
	}
}

func (p *Proxy) DestroyConn(w http.ResponseWriter, r *http.Request) {
	p.client.DestroyConn()
	w.Write([]byte("success\n"))
}

func (p *Proxy) Stats(w http.ResponseWriter, r *http.Request) {
	alldata := stats.GetAllMetricsData()
	b, _ := json.MarshalIndent(alldata, "", "\t")
	w.Write(b)
}
