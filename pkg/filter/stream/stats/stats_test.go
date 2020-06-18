package stats

import (
	"context"
	"net"
	"testing"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/istio/utils"
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
		config []MetricConfig
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
		config []MetricConfig
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
