package stats

import (
	"context"
	"net"
	"testing"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/istio/istio1106/istio/utils"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

func TestParseStreamStatsFilter(t *testing.T) {
	m := map[string]interface{}{}
	_, err := CreateStatsFilterFactory(m)
	if err != nil {
		t.Error("parse stream stats filter")
		return
	}
}

func TestStatsFilterLog(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	type args struct {
		reqHeaders  api.HeaderMap
		respHeaders api.HeaderMap
		requestInfo api.RequestInfo
		buf         buffer.IoBuffer
		trailers    api.HeaderMap
	}
	tests := []struct {
		name   string
		config []*MetricConfig
		args   args
	}{
		{
			name:   "default metric, empty data",
			config: defaultMetricConfig,
			args: args{
				reqHeaders:  protocol.CommonHeader{},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{},
				buf:         buffer.NewIoBuffer(0),
				trailers:    protocol.CommonHeader{},
			},
		},
		{
			name:   "default metric, simple data",
			config: defaultMetricConfig,
			args: args{
				reqHeaders:  protocol.CommonHeader{},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{
					startTime:              now,
					endTime:                now,
					protocol:               protocol.HTTP1,
					downstreamLocalAddress: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 80},
					upstreamHost: &MockHostInfo{
						addressString: "10.0.0.2:80",
					},
				},
				buf:      buffer.NewIoBuffer(0),
				trailers: protocol.CommonHeader{},
			},
		},
		{
			name:   "default metric, simple data with x-istio-attributes",
			config: defaultMetricConfig,
			args: args{
				reqHeaders: protocol.CommonHeader{
					utils.KIstioAttributeHeader: `Cj8KGGRlc3RpbmF0aW9uLnNlcnZpY2UuaG9zdBIjEiFodHRwYmluLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwKPQoXZGVzdGluYXRpb24uc2VydmljZS51aWQSIhIgaXN0aW86Ly9kZWZhdWx0L3NlcnZpY2VzL2h0dHBiaW4KKgodZGVzdGluYXRpb24uc2VydmljZS5uYW1lc3BhY2USCRIHZGVmYXVsdAolChhkZXN0aW5hdGlvbi5zZXJ2aWNlLm5hbWUSCRIHaHR0cGJpbgo6Cgpzb3VyY2UudWlkEiwSKmt1YmVybmV0ZXM6Ly9zbGVlcC03YjlmOGJmY2QtMmRqeDUuZGVmYXVsdAo6ChNkZXN0aW5hdGlvbi5zZXJ2aWNlEiMSIWh0dHBiaW4uZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbA==`,
				},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{
					startTime:              now,
					endTime:                now,
					protocol:               protocol.HTTP1,
					downstreamLocalAddress: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 80},
					upstreamHost: &MockHostInfo{
						addressString: "10.0.0.2:80",
					},
				},
				buf:      buffer.NewIoBuffer(0),
				trailers: protocol.CommonHeader{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, err := newMetrics(defaultMetricConfig, defaultMetricDefinition)
			if err != nil {
				t.Fatal(err)
				return
			}

			filter := newStatsFilter(ctx, "istio", ms)
			filter.OnReceive(ctx, tt.args.reqHeaders, tt.args.buf, tt.args.trailers)
			filter.Log(ctx, tt.args.reqHeaders, tt.args.respHeaders, tt.args.requestInfo)
		})
	}
}

func BenchmarkStatsFilterLog(b *testing.B) {
	now := time.Now()
	ctx := context.Background()
	type args struct {
		reqHeaders  api.HeaderMap
		respHeaders api.HeaderMap
		requestInfo api.RequestInfo
		buf         buffer.IoBuffer
		trailers    api.HeaderMap
	}
	tests := []struct {
		name   string
		config []*MetricConfig
		args   args
	}{
		{
			name:   "default metric, empty data",
			config: defaultMetricConfig,
			args: args{
				reqHeaders:  protocol.CommonHeader{},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{},
				buf:         buffer.NewIoBuffer(0),
				trailers:    protocol.CommonHeader{},
			},
		},
		{
			name:   "default metric, simple data",
			config: defaultMetricConfig,
			args: args{
				reqHeaders:  protocol.CommonHeader{},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{
					startTime:              now,
					endTime:                now,
					protocol:               protocol.HTTP1,
					downstreamLocalAddress: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 80},
					upstreamHost: &MockHostInfo{
						addressString: "10.0.0.2:80",
					},
				},
				buf:      buffer.NewIoBuffer(0),
				trailers: protocol.CommonHeader{},
			},
		},
		{
			name:   "default metric, simple data with x-istio-attributes",
			config: defaultMetricConfig,
			args: args{
				reqHeaders: protocol.CommonHeader{
					utils.KIstioAttributeHeader: `Cj8KGGRlc3RpbmF0aW9uLnNlcnZpY2UuaG9zdBIjEiFodHRwYmluLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwKPQoXZGVzdGluYXRpb24uc2VydmljZS51aWQSIhIgaXN0aW86Ly9kZWZhdWx0L3NlcnZpY2VzL2h0dHBiaW4KKgodZGVzdGluYXRpb24uc2VydmljZS5uYW1lc3BhY2USCRIHZGVmYXVsdAolChhkZXN0aW5hdGlvbi5zZXJ2aWNlLm5hbWUSCRIHaHR0cGJpbgo6Cgpzb3VyY2UudWlkEiwSKmt1YmVybmV0ZXM6Ly9zbGVlcC03YjlmOGJmY2QtMmRqeDUuZGVmYXVsdAo6ChNkZXN0aW5hdGlvbi5zZXJ2aWNlEiMSIWh0dHBiaW4uZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbA==`,
				},
				respHeaders: protocol.CommonHeader{},
				requestInfo: &MockRequestInfo{
					startTime:              now,
					endTime:                now,
					protocol:               protocol.HTTP1,
					downstreamLocalAddress: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 80},
					upstreamHost: &MockHostInfo{
						addressString: "10.0.0.2:80",
					},
				},
				buf:      buffer.NewIoBuffer(0),
				trailers: protocol.CommonHeader{},
			},
		},
	}
	for _, tt := range tests {
		ms, err := newMetrics(defaultMetricConfig, defaultMetricDefinition)
		if err != nil {
			b.Fatal(err)
			return
		}
		filter := newStatsFilter(ctx, "istio", ms)
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				filter.OnReceive(ctx, tt.args.reqHeaders, tt.args.buf, tt.args.trailers)
				filter.Log(ctx, tt.args.reqHeaders, tt.args.respHeaders, tt.args.requestInfo)
			}
		})
	}
}

// MockRequestInfo
type MockRequestInfo struct {
	protocol                 api.ProtocolName
	startTime                time.Time
	endTime                  time.Time
	responseFlag             api.ResponseFlag
	upstreamHost             api.HostInfo
	requestReceivedDuration  time.Duration
	requestFinishedDuration  time.Duration
	responseReceivedDuration time.Duration
	processTimeDuration      time.Duration
	bytesSent                uint64
	bytesReceived            uint64
	responseCode             int
	localAddress             string
	downstreamLocalAddress   net.Addr
	downstreamRemoteAddress  net.Addr
	isHealthCheckRequest     bool
	routerRule               api.RouteRule
}

func (r *MockRequestInfo) StartTime() time.Time {
	return r.startTime
}

func (r *MockRequestInfo) SetStartTime() {
	r.startTime = time.Now()
}

func (r *MockRequestInfo) RequestReceivedDuration() time.Duration {
	return r.requestReceivedDuration
}

func (r *MockRequestInfo) SetRequestReceivedDuration(t time.Time) {
	r.requestReceivedDuration = t.Sub(r.startTime)
}

func (r *MockRequestInfo) ResponseReceivedDuration() time.Duration {
	return r.responseReceivedDuration
}

func (r *MockRequestInfo) SetResponseReceivedDuration(t time.Time) {
	r.responseReceivedDuration = t.Sub(r.startTime)
}

func (r *MockRequestInfo) RequestFinishedDuration() time.Duration {
	return r.requestFinishedDuration
}

func (r *MockRequestInfo) SetRequestFinishedDuration(t time.Time) {
	r.requestFinishedDuration = t.Sub(r.startTime)

}

func (r *MockRequestInfo) ProcessTimeDuration() time.Duration {
	return r.processTimeDuration
}

func (r *MockRequestInfo) SetProcessTimeDuration(d time.Duration) {
	r.processTimeDuration = d
}

func (r *MockRequestInfo) BytesSent() uint64 {
	return r.bytesSent
}

func (r *MockRequestInfo) SetBytesSent(bytesSent uint64) {
	r.bytesSent = bytesSent
}

func (r *MockRequestInfo) BytesReceived() uint64 {
	return r.bytesReceived
}

func (r *MockRequestInfo) SetBytesReceived(bytesReceived uint64) {
	r.bytesReceived = bytesReceived
}

func (r *MockRequestInfo) Protocol() api.ProtocolName {
	return r.protocol
}

func (r *MockRequestInfo) SetProtocol(p api.ProtocolName) {
	r.protocol = p
}

func (r *MockRequestInfo) ResponseCode() int {
	return r.responseCode
}

func (r *MockRequestInfo) SetResponseCode(code int) {
	r.responseCode = code
}

func (r *MockRequestInfo) Duration() time.Duration {
	return r.endTime.Sub(r.startTime)
}

func (r *MockRequestInfo) GetResponseFlag(flag api.ResponseFlag) bool {
	return r.responseFlag&flag != 0
}

func (r *MockRequestInfo) SetResponseFlag(flag api.ResponseFlag) {
	r.responseFlag |= flag
}

func (r *MockRequestInfo) UpstreamHost() api.HostInfo {
	return r.upstreamHost
}

func (r *MockRequestInfo) OnUpstreamHostSelected(host api.HostInfo) {
	r.upstreamHost = host
}

func (r *MockRequestInfo) UpstreamLocalAddress() string {
	return r.localAddress
}

func (r *MockRequestInfo) SetUpstreamLocalAddress(addr string) {
	r.localAddress = addr
}

func (r *MockRequestInfo) IsHealthCheck() bool {
	return r.isHealthCheckRequest
}

func (r *MockRequestInfo) SetHealthCheck(isHc bool) {
	r.isHealthCheckRequest = isHc
}

func (r *MockRequestInfo) DownstreamLocalAddress() net.Addr {
	return r.downstreamLocalAddress
}

func (r *MockRequestInfo) SetDownstreamLocalAddress(addr net.Addr) {
	r.downstreamLocalAddress = addr
}

func (r *MockRequestInfo) DownstreamRemoteAddress() net.Addr {
	return r.downstreamRemoteAddress
}

func (r *MockRequestInfo) SetDownstreamRemoteAddress(addr net.Addr) {
	r.downstreamRemoteAddress = addr
}

func (r *MockRequestInfo) RouteEntry() api.RouteRule {
	return r.routerRule
}

func (r *MockRequestInfo) SetRouteEntry(routerRule api.RouteRule) {
	r.routerRule = routerRule
}

type MockHostInfo struct {
	supportTLS    bool
	health        bool
	weight        uint32
	healthFlag    api.HealthFlag
	metadata      api.Metadata
	hostname      string
	addressString string
}

func (h *MockHostInfo) Hostname() string {
	return h.hostname
}

func (h *MockHostInfo) Metadata() api.Metadata {
	return h.metadata
}

func (h *MockHostInfo) AddressString() string {
	return h.addressString
}

func (h *MockHostInfo) Weight() uint32 {
	return h.weight
}

func (h *MockHostInfo) SupportTLS() bool {
	return h.supportTLS
}

func (h *MockHostInfo) ClearHealthFlag(flag api.HealthFlag) {

}

func (h *MockHostInfo) ContainHealthFlag(flag api.HealthFlag) bool {
	return false
}

func (h *MockHostInfo) SetHealthFlag(flag api.HealthFlag) {

}

func (h *MockHostInfo) HealthFlag() api.HealthFlag {
	return h.healthFlag
}

func (h *MockHostInfo) Health() bool {
	return h.health
}
