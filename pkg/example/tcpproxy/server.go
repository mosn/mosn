package main

import (
	"net/http"
	"time"
	"fmt"
	_ "net/http/pprof"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	stopChan := make(chan bool)

	go func() {
		// mesh
		cmf := &clusterManagerFilter{}
		srv := server.NewServer(&proxy.TcpProxyFilterConfigFactory{
			Proxy: tcpProxyConfig(),
		}, cmf)
		srv.AddListener(tcpListener())
		cmf.cccb.UpdateClusterConfig(clusters())
		cmf.chcb.UpdateClusterHost(TestCluster, 0, hosts("11.162.169.38:80"))

		srv.Start()

		select {
		case <-stopChan:
			srv.Close()
		}
	}()

	select {
	case <-time.After(time.Second * 1800):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}
}
