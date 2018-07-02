package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"

	"crypto/tls"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/router/basic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/stream/http2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"golang.org/x/net/http2"
	"io/ioutil"
)

const (
	RealServerAddr  = "127.0.0.1:8088"
	MeshServerAddr  = "127.0.0.1:2044"
	TestCluster     = "tstCluster"
	TestListenerRPC = "tstListener"
)

func main() {
	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	log.InitDefaultLogger("", log.DEBUG)

	stopChan := make(chan bool)
	meshReadyChan := make(chan bool)

	go func() {
		// upstream
		server := &http.Server{
			Addr:         ":8080",
			Handler:      &serverHandler{},
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		s2 := &http2.Server{
			IdleTimeout: 1 * time.Minute,
		}

		http2.ConfigureServer(server, s2)
		l, _ := net.Listen("tcp", RealServerAddr)
		defer l.Close()

		for {
			rwc, err := l.Accept()
			if err != nil {
				fmt.Println("accept err:", err)
				continue
			}
			go s2.ServeConn(rwc, &http2.ServeConnOpts{BaseConfig: server})
		}
	}()

	select {
	case <-time.After(2 * time.Second):
	}

	go func() {
		//  mesh
		cmf := &clusterManagerFilterRPC{}
		cm := cluster.NewClusterManager(nil, nil, nil, false)

		//RPC
		srv := server.NewServer(&server.Config{}, cmf, cm)

		srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
			Proxy: genericProxyConfig(),
		}, nil)
		cmf.cccb.UpdateClusterConfig(clustersrpc())
		cmf.chcb.UpdateClusterHost(TestCluster, 0, rpchosts())

		meshReadyChan <- true

		srv.Start() //开启连接

		select {
		case <-stopChan:
			srv.Close()
		}
	}()

	go func() {
		select {
		case <-meshReadyChan:
			// client
			tr := &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(netw, addr)
				},
			}

			httpClient := http.Client{Transport: tr}
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/", MeshServerAddr), nil)
			req.Header.Add("service", "tst")
			resp, err := httpClient.Do(req)

			if err != nil {
				fmt.Printf("[CLIENT]receive err %s", err)
				fmt.Println()
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				fmt.Printf("[CLIENT]receive err %s", err)
				fmt.Println()
				return
			}

			fmt.Printf("[CLIENT]receive data %s", body)
			fmt.Println()
		}
	}()

	select {
	case <-time.After(time.Second * 10):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}
}

type serverHandler struct {
}

func (sh *serverHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ShowRequestInfoHandler(w, req)
}

func ShowRequestInfoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s", r.URL)
	fmt.Println()

	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "RequestURI: %q\n", r.RequestURI)
	fmt.Fprintf(w, "URL: %#v\n", r.URL)
	fmt.Fprintf(w, "Body.ContentLength: %d (-1 means unknown)\n", r.ContentLength)
	fmt.Fprintf(w, "Close: %v (relevant for HTTP/1 only)\n", r.Close)
	fmt.Fprintf(w, "TLS: %#v\n", r.TLS)
	fmt.Fprintf(w, "\nHeaders:\n")

	r.Header.Write(w)
}

func genericProxyConfig() *v2.Proxy {
	proxyConfig := &v2.Proxy{
		DownstreamProtocol: string(protocol.Http2),
		UpstreamProtocol:   string(protocol.Http2),
	}

	header := v2.HeaderMatcher{
		Name:  "service",
		Value: "com.alipay.rpc.common.service.facade.SampleService:1.0",
	}

	routerV2 := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header},
		},

		Route: v2.RouteAction{
			ClusterName: TestCluster,
		},
	}

	proxyConfig.VirtualHosts = append(proxyConfig.VirtualHosts, &v2.VirtualHost{
		Name:    "testSofaRoute",
		Domains: []string{"*"},
		Routers: []v2.Router{routerV2},
	})

	return proxyConfig
}

func rpcProxyListener() *v2.ListenerConfig {
	addr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)

	return &v2.ListenerConfig{
		Name:                    TestListenerRPC,
		Addr:                    addr,
		BindToPort:              true,
		PerConnBufferLimitBytes: 1024 * 32,
		LogPath:                 "",
		LogLevel:                uint8(log.DEBUG),
		DisableConnIo:           true,
	}
}

func rpchosts() []v2.Host {
	var hosts []v2.Host

	hosts = append(hosts, v2.Host{
		Address: RealServerAddr,
		Weight:  100,
	})

	return hosts
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
		Name:                 TestCluster,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 32 * 1024,
	})

	return configs
}
