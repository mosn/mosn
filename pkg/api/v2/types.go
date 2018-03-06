package v2

import "net"

type Metadata struct {
}

type ClusterType string

const (
	STATIC_CLUSTER ClusterType = "STATIC"
	SIMPLE_CLUSTER ClusterType = "SIMPLE"
)

type LbType string

const (
	LB_RANDOM     LbType = "LB_RANDOM"
	LB_ROUNDROBIN LbType = "LB_ROUNDROBIN"
)

type Cluster struct {
	Name                 string
	ClusterType          ClusterType
	LbType               LbType
	MaxRequestPerConn    uint64
	ConnBufferLimitBytes int
}

type Host struct {
	Address  string
	Hostname string
	Weight   uint32
}

type ListenerConfig struct {
	Name                                  string
	Addr                                  string
	ListenerTag                           uint64
	ListenerScope                         string
	BindToPort                            bool
	ConnBufferLimitBytes                  uint32
	HandOffRestoredDestinationConnections bool
}

type TcpRoute struct {
	Cluster          string
	SourceAddrs      []net.Addr
	DestinationAddrs []net.Addr
}

type TcpProxy struct {
	Routes []*TcpRoute
}

type RpcRoute struct {
	Name string
	Service string
	Cluster string
}

type RpcProxy struct {
	Routes []*RpcRoute
}
