package network

import (
	"time"
	"net"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// RequestInfo
type requestInfo struct {
}

func NewRequestInfo() types.RequestInfo {
	return &requestInfo{}
}

func (r *requestInfo) SetResponseFlag(flag types.ResponseFlag) {

}

func (r *requestInfo) OnUpstreamHostSelected(host types.HostInfo) {

}

func (r *requestInfo) StartTime() time.Time {
	return time.Now()
}

func (r *requestInfo) RequestReceivedDuration() int {
	return 0
}

func (r *requestInfo) SetRequestReceivedDuration(duration int) {

}

func (r *requestInfo) ResponseReceivedDuration() int {
	return 0
}

func (r *requestInfo) SetResponseReceivedDuration(duration int) {

}

func (r *requestInfo) BytesSent() uint64 {
	return 0
}

func (r *requestInfo) SetBytesSent(bytesSent uint64) {

}

func (r *requestInfo) BytesReceived() uint64 {
	return 0
}

func (r *requestInfo) SetBytesReceived(bytesReceived uint64) {

}

func (r *requestInfo) Protocol() string {
	return ""
}

func (r *requestInfo) SetProtocol(protocol string) {

}

func (r *requestInfo) ResponseCode() uint32 {
	return 0
}

func (r *requestInfo) Duration() int {
	return 0
}

func (r *requestInfo) GetResponseFlag(flag int) {

}

func (r *requestInfo) UpstreamHost() types.HostInfo {
	return nil
}

func (r *requestInfo) UpstreamLocalAddress() net.Addr {
	return nil
}

func (r *requestInfo) HealthCheck() bool {
	return false
}

func (r *requestInfo) SetHealthCheck(isHc bool) {}

func (r *requestInfo) DownstreamLocalAddress() net.Addr {
	return nil
}

func (r *requestInfo) SetDownstreamLocalAddress(addr net.Addr) {

}

func (r *requestInfo) DownstreamRemoteAddress() net.Addr {
	return nil
}

func (r *requestInfo) SetDownstreamRemoteAddress(addr net.Addr) {

}

func (r *requestInfo) RouteEntry() types.RouteRule {
	return nil
}
