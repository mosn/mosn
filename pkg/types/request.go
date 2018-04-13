package types

import (
	"time"
	"net"
)

type ResponseFlag int

const (
	NoHealthyUpstream             ResponseFlag = 0x2
	UpstreamRequestTimeout        ResponseFlag = 0x4
	UpstreamLocalReset            ResponseFlag = 0x8
	UpstreamRemoteReset           ResponseFlag = 0x10
	UpstreamConnectionFailure     ResponseFlag = 0x20
	UpstreamConnectionTermination ResponseFlag = 0x40
	UpstreamOverflow              ResponseFlag = 0x80
	NoRouteFound                  ResponseFlag = 0x100
	DelayInjected                 ResponseFlag = 0x200
	FaultInjected                 ResponseFlag = 0x400
	RateLimited                   ResponseFlag = 0x800
)

type RequestInfo interface {
	StartTime() time.Time

	RequestReceivedDuration() time.Duration

	SetRequestReceivedDuration(time time.Time)

	ResponseReceivedDuration() time.Duration

	SetResponseReceivedDuration(time time.Time)

	BytesSent() uint64

	SetBytesSent(bytesSent uint64)

	BytesReceived() uint64

	SetBytesReceived(bytesReceived uint64)

	Protocol() Protocol

	ResponseCode() uint32

	Duration() time.Duration

	GetResponseFlag(flag ResponseFlag) bool

	SetResponseFlag(flag ResponseFlag)

	UpstreamHost() HostInfo

	OnUpstreamHostSelected(host HostInfo)

	UpstreamLocalAddress() net.Addr

	SetUpstreamLocalAddress(localAddress net.Addr)

	IsHealthCheck() bool

	SetHealthCheck(isHc bool)

	DownstreamLocalAddress() net.Addr

	SetDownstreamLocalAddress(addr net.Addr)

	DownstreamRemoteAddress() net.Addr

	SetDownstreamRemoteAddress(addr net.Addr)

	RouteEntry() RouteRule

	SetRouteEntry(routerRule RouteRule)
}
