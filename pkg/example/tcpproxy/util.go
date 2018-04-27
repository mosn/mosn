package main

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net"
)

const (
	TestListener   = "tstListener"
	MeshServerAddr = "127.0.0.1:2048"
)

type clusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func tcpProxyConfig() *v2.TcpProxy {
	tcpProxyConfig := &v2.TcpProxy{}
	tcpProxyConfig.Routes = append(tcpProxyConfig.Routes, &v2.TcpRoute{
		Cluster: TestCluster,
	})
	tcpProxyConfig.AccessLogs = append(tcpProxyConfig.AccessLogs, &v2.AccessLog{})

	return tcpProxyConfig
}

func tcpListener() *v2.ListenerConfig {
	addr, _ := net.ResolveTCPAddr("tcp", MeshServerAddr)

	return &v2.ListenerConfig{
		Name:                    TestListener,
		Addr:                    addr,
		BindToPort:              true,
		PerConnBufferLimitBytes: 1024 * 32,
		LogPath:                 "",
		LogLevel:                uint8(log.DEBUG),
	}
}

func clusters() []v2.Cluster {
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

func hosts(host string) []v2.Host {
	var hosts []v2.Host

	if host == "" {
		host = RealServerAddr
	}

	hosts = append(hosts, v2.Host{
		Address: host,
		Weight:  100,
	})

	return hosts
}
