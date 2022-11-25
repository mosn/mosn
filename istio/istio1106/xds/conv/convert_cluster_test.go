package conv

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
	"testing"
	"time"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	duration "github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"mosn.io/mosn/pkg/config/v2"
)

func TestConvertClustersConfig_OriginalDst(t *testing.T) {
	smeta, err := structpb.NewStruct(map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"host":      "ratings.default.svc.cluster.local",
				"name":      "ratings",
				"namespace": "default",
			},
		},
	})
	require.Nil(t, err)
	inboundCluster := &envoy_config_cluster_v3.Cluster{
		Name: "inbound|9080||",
		ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
			Type: envoy_config_cluster_v3.Cluster_ORIGINAL_DST,
		},
		ConnectTimeout: &duration.Duration{
			Seconds: 10,
		},
		LbPolicy: envoy_config_cluster_v3.Cluster_CLUSTER_PROVIDED,
		CircuitBreakers: &envoy_config_cluster_v3.CircuitBreakers{
			Thresholds: []*envoy_config_cluster_v3.CircuitBreakers_Thresholds{
				{
					MaxConnections: &wrappers.UInt32Value{
						Value: 4294967295,
					},
					MaxPendingRequests: &wrappers.UInt32Value{
						Value: 4294967295,
					},
					MaxRequests: &wrappers.UInt32Value{
						Value: 4294967295,
					},
					MaxRetries: &wrappers.UInt32Value{
						Value: 4294967295,
					},
					TrackRemaining: true,
				},
			},
		},
		CleanupInterval: &duration.Duration{
			Seconds: 60,
		},
		UpstreamBindConfig: &envoy_config_core_v3.BindConfig{
			SourceAddress: &envoy_config_core_v3.SocketAddress{
				Address: "127.0.0.6",
				PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
					PortValue: 0,
				},
			},
		},
		Metadata: &envoy_config_core_v3.Metadata{
			FilterMetadata: map[string]*structpb.Struct{
				"istio": smeta,
			},
		},
	}
	outBoundCluster := &envoy_config_cluster_v3.Cluster{
		Name: "outbound|9100||node-exporter.arms-prom.svc.cluster.local",
		ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
			Type: envoy_config_cluster_v3.Cluster_ORIGINAL_DST,
		},
		ConnectTimeout: &duration.Duration{
			Seconds: 10,
		},
		LbPolicy: envoy_config_cluster_v3.Cluster_CLUSTER_PROVIDED,
	}
	clusterConfigs := ConvertClustersConfig([]*envoy_config_cluster_v3.Cluster{
		inboundCluster,
		outBoundCluster,
	})
	require.Len(t, clusterConfigs, 2)
	inBoundClusterConfig := clusterConfigs[0]
	require.Equal(t, "inbound|9080||", inBoundClusterConfig.Name)
	require.Equal(t, v2.ORIGINALDST_CLUSTER, inBoundClusterConfig.ClusterType)
	require.Equal(t, v2.LB_ORIGINAL_DST, inBoundClusterConfig.LbType)
	require.Equal(t, 10*time.Second, inBoundClusterConfig.ConnectTimeout.Duration)
	require.True(t, inBoundClusterConfig.LBOriDstConfig.ReplaceLocal)

	outBoundClusterConfig := clusterConfigs[1]

	require.Equal(t, "outbound|9100||node-exporter.arms-prom.svc.cluster.local", outBoundClusterConfig.Name)
	require.Equal(t, v2.ORIGINALDST_CLUSTER, outBoundClusterConfig.ClusterType)
	require.Equal(t, v2.LB_ORIGINAL_DST, outBoundClusterConfig.LbType)
	require.False(t, outBoundClusterConfig.LBOriDstConfig.ReplaceLocal)

	/*
		m := jsonpb.Marshaler{
			OrigName: true,
		}
		d, _ := m.MarshalToString(inboundCluster)
		fmt.Println(d)
	*/
}

func Test_convertHealthChecks(t *testing.T) {
	type args struct {
		serviceName     string
		xdsHealthChecks []*envoy_config_core_v3.HealthCheck
	}
	tests := []struct {
		name string
		args args
		want v2.HealthCheck
	}{
		{
			name: "convert health checks config",
			args: args{serviceName: "mosn-test", xdsHealthChecks: []*envoy_config_core_v3.HealthCheck{
				{HealthyThreshold: &wrappers.UInt32Value{Value: 1},
					UnhealthyThreshold: &wrappers.UInt32Value{Value: 1},
					Timeout:            durationpb.New(2 * time.Second),
					Interval:           durationpb.New(2 * time.Second),
					IntervalJitter:     durationpb.New(time.Second * 2)},
			},
			}, want: v2.HealthCheck{
				HealthCheckConfig: v2.HealthCheckConfig{
					ServiceName:        "mosn-test",
					HealthyThreshold:   1,
					UnhealthyThreshold: 1,
				},
				Timeout:        time.Second * 2,
				Interval:       time.Second * 2,
				IntervalJitter: time.Second * 2,
			},
		},
		{
			name: "don't convert health checks config",
			args: args{
				serviceName:     "mosn-test",
				xdsHealthChecks: nil,
			}, want: v2.HealthCheck{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, convertHealthChecks(tt.args.serviceName, tt.args.xdsHealthChecks), "convertHealthChecks(%v, %v)", tt.args.serviceName, tt.args.xdsHealthChecks)
		})
	}
}
