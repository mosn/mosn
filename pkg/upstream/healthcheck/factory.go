package healthcheck


import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func NewSessionFactory(c *healthChecker,host types.Host) types.HealthCheckSession {
	
	var sfhcs types.HealthCheckSession
	
	switch host.ClusterInfo().HealthCheckProtocol(){
		case sofarpc.SofaRpc:
			log.DefaultLogger.Debugf("Add sofa health check session, remote host address = %s",host.AddressString())
			sfhc := NewSofaRpcHealthCheckWithHC(c,sofarpc.BOLT_V1)
			sfhcs = sfhc.NewSofaRpcHealthCheckSession(nil,host)
			
			// todo support other protocol
	}
	
	return sfhcs
}