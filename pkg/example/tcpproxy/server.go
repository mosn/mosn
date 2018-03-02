package main

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"time"
	"fmt"
)

func main() {
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
