package types

import (
	"net"
	"time"
)

type ResponseFlag int

// Some Response Flags
const (
	// no healthy upstream found
	NoHealthyUpstream ResponseFlag = 0x2
	// Upstream Request timeout
	UpstreamRequestTimeout ResponseFlag = 0x4
	// local reset
	UpstreamLocalReset ResponseFlag = 0x8
	// upstream reset
	UpstreamRemoteReset ResponseFlag = 0x10
	// connect upstream failure
	UpstreamConnectionFailure ResponseFlag = 0x20
	// upstream terminate connection
	UpstreamConnectionTermination ResponseFlag = 0x40
	// upstream's connection overflow
	UpstreamOverflow ResponseFlag = 0x80
	// no route found
	NoRouteFound ResponseFlag = 0x100
	// inject delay
	DelayInjected ResponseFlag = 0x200
	// inject fault
	FaultInjected ResponseFlag = 0x400
	// rate limited
	RateLimited ResponseFlag = 0x800
)

type RequestInfo interface {
	// get request's arriving time
	StartTime() time.Time

	// get duration between request arriving and request resend to upstream
	RequestReceivedDuration() time.Duration

	// set duration between request arriving and request resend to upstream
	SetRequestReceivedDuration(time time.Time)

	// get duration between request arriving and response sending
	ResponseReceivedDuration() time.Duration

	// set duration between request arriving and response sending
	SetResponseReceivedDuration(time time.Time)

	// get bytes sent
	BytesSent() uint64

	// set bytes sent
	SetBytesSent(bytesSent uint64)

	// get bytes received
	BytesReceived() uint64

	// set  bytes received
	SetBytesReceived(bytesReceived uint64)

	// get request's protocol type
	Protocol() Protocol

	// get request's response code
	ResponseCode() uint32

	// get duration since request's starting time
	Duration() time.Duration

	// get request's response flag
	GetResponseFlag(flag ResponseFlag) bool

	// set request's response flag
	SetResponseFlag(flag ResponseFlag)

	// get upstream selected
	UpstreamHost() HostInfo

	// set upstream selected
	OnUpstreamHostSelected(host HostInfo)

	// get upstream's local address
	UpstreamLocalAddress() net.Addr

	// set upstream's local address
	SetUpstreamLocalAddress(localAddress net.Addr)

	// is health check
	IsHealthCheck() bool

	// set health check
	SetHealthCheck(isHc bool)

	// get downstream's local address
	DownstreamLocalAddress() net.Addr

	// set downstream's local address
	SetDownstreamLocalAddress(addr net.Addr)

	// get downstream's local address
	DownstreamRemoteAddress() net.Addr

	// set downstream's local address
	SetDownstreamRemoteAddress(addr net.Addr)

	// get route rule
	RouteEntry() RouteRule

	// set route rule
	SetRouteEntry(routerRule RouteRule)
}
