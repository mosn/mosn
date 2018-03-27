package v2

import (
	"net"
	"time"
)

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

type AccessLog struct {
	Path   string
	Format string
	// todo: add log filters
}

type TcpRoute struct {
	Cluster          string
	SourceAddrs      []net.Addr
	DestinationAddrs []net.Addr
}

type TcpProxy struct {
	Routes     []*TcpRoute
	AccessLogs []*AccessLog
}

type RpcRoute struct {
	Name    string
	Service string
	Cluster string
}

type RpcProxy struct {
	Routes []*RpcRoute
}

type FaultInject struct {
	DelayPercent  uint32
	DelayDuration uint64
}

type Proxy struct {
	DownstreamProtocol string
	UpstreamProtocol   string
	Routes             []*BasicServiceRoute
	AccessLogs         []*AccessLog
}

type BasicServiceRoute struct {
	Name          string
	Service       string
	Cluster       string
	GlobalTimeout time.Duration
	RetryPolicy   *RetryPolicy
}

type RetryPolicy struct {
	RetryOn      bool
	RetryTimeout time.Duration
	NumRetries   int
}

type HealthCheck struct {
	Timeout            time.Duration
	HealthyThreshold   uint32
	UnhealthyThreshold uint32
	Interval           time.Duration
	IntervalJitter     time.Duration
	CheckPath          string
	ServiceName        string
}
