package healthcheck

import (
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

// use for hearth-beat starting for sofa bolt in the same codecClient
// for bolt heartbeat, timeout: 90s interval: 15s
func StartSofaHeartBeat(timeout time.Duration, interval time.Duration, hostAddr string,
	codecClient stream.CodecClient, nameHB string, pro sofarpc.ProtocolType) types.HealthCheckSession {

	hcV2 := v2.HealthCheck{
		Timeout:     timeout,
		Interval:    interval,
		ServiceName: nameHB,
	}

	hostV2 := v2.Host{
		Address: hostAddr,
	}

	host := cluster.NewHost(hostV2, nil)
	baseHc := newHealthChecker(hcV2)

	hc := newSofaRpcHealthCheckerWithBaseHealthChecker(baseHc, pro)
	hcs := hc.newSofaRpcHealthCheckSession(codecClient, host)
	hcs.Start()

	return hcs
}
