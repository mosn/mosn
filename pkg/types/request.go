package types

import (
	"time"
	"net"
)

type RequestInfo interface {
	SetResponseFlag(flag ResponseFlag)

	OnUpstreamHostSelected(host HostInfo)

	StartTime() time.Time

	RequestReceivedDuration() int

	SetRequestReceivedDuration(duration int)

	ResponseReceivedDuration() int

	SetResponseReceivedDuration(duration int)

	BytesSent() uint64

	SetBytesSent(bytesSent uint64)

	BytesReceived() uint64

	SetBytesReceived(bytesReceived uint64)

	Protocol() string

	SetProtocol(protocol string)

	ResponseCode() uint32

	Duration() int

	GetResponseFlag(flag int)

	UpstreamHost() HostInfo

	UpstreamLocalAddress() net.Addr

	HealthCheck() bool

	SetHealthCheck(isHc bool)

	DownstreamLocalAddress() net.Addr

	SetDownstreamLocalAddress(addr net.Addr)

	DownstreamRemoteAddress() net.Addr

	SetDownstreamRemoteAddress(addr net.Addr)

	RouteEntry() RouteRule
}
