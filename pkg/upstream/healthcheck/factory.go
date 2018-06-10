package healthcheck


import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func NewSessionFactory(c *healthChecker,host types.Host) types.HealthCheckSession {
	
	if host.ClusterInfo().HealthCheckProtocol() == types.SofaRpc {
		log.DefaultLogger.Debugf("Add sofa health check session, remote host address = %s",host.AddressString())
		sfhc := NewSofaRpcHealthCheckWithHC(c,sofarpc.BOLT_V1)
		sfhcs := sfhc.NewSofaRpcHealthCheckSession(nil,host)
		return sfhcs
	}
	
	// todo support other protocol
	return nil
}
