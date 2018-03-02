package main

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

const (
	TestListener = "tstListener"
	MeshServerAddr = "127.0.0.1:2048"
)

type clusterManagerFilter struct {
	cccb server.ClusterConfigFactoryCb
	chcb server.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilter) OnCreated(cccb server.ClusterConfigFactoryCb, chcb server.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

func tcpProxyConfig() *v2.TcpProxy {
	tcpProxyConfig := &v2.TcpProxy{}
	tcpProxyConfig.Routes = append(tcpProxyConfig.Routes, &v2.TcpRoute{
		Cluster: TestCluster,
	})

	return tcpProxyConfig
}

func tcpListener() v2.ListenerConfig {
	return v2.ListenerConfig{
		Name:                 TestListener,
		Addr:                 MeshServerAddr,
		BindToPort:           true,
		ConnBufferLimitBytes: 1024 * 32,
	}
}

func clusters() []v2.Cluster {
	var configs []v2.Cluster
	configs = append(configs, v2.Cluster{
		Name:                 TestCluster,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 16 * 1026,
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
