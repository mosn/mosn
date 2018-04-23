package main

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	var srv server.Server

	go func() {
		// mesh
		cmf := &clusterManagerFilter{}
		srv = server.NewServer(nil, cmf)
		srv.AddListener(tcpListener(), &proxy.TcpProxyFilterConfigFactory{
			Proxy: tcpProxyConfig(),
		}, nil)
		cmf.cccb.UpdateClusterConfig(clusters())
		cmf.chcb.UpdateClusterHost(TestCluster, 0, hosts("11.162.169.38:80"))

		srv.Start()
	}()

	select {
	case <-time.After(time.Second * 100):
		srv.Close()
		fmt.Println("[MAIN]closing..")
	}
}
