package v2

import "net"

type Metadata struct {
}

type ClusterType string

const (
	STATIC ClusterType = "STATIC"
	SIMPLE ClusterType = "SIMPLE"
)

type LbType string

const (
	RANDOM      LbType = "RANDOM"
	ROUND_ROBIN LbType = "ROUND_ROBIN"
)

type Cluster struct {
	Name                 string
	ClusterType          ClusterType
	LbType               LbType
	MaxRequestPerConn    uint64
	ConnBufferLimitBytes int
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
