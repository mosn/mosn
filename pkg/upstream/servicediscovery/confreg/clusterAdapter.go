package registry

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
	"strings"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

func init() {
	cf := &confregAdaptor{
		ca:&cluster.ClusterAdap,
	}
	servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(cf)
}

type confregAdaptor struct {
	ca    *cluster.ClusterAdapter
}

func (cf *confregAdaptor) RegisterUpdateMethod() {
	servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(cf)
}

func (cf *confregAdaptor) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {

	log.StartLogger.Debugf("[Call back by confreg]", zoneServers)

	dataId = dataId[:len(dataId)-8]
	serviceName := dataId

	log.StartLogger.Debugf("[Service Name]", serviceName)
	var hosts []v2.Host
	for _, val := range zoneServers {
		for _, v := range val {

			idx := strings.Index(v, "?")
			if idx > 0 {
				ipaddress := v[:idx]
				hosts = append(hosts, v2.Host{
					Address: ipaddress,
				})
				log.StartLogger.Debugf("IP_ADDR", ipaddress)
			}
		}
	}
	//get clusterName
	clusterName := types.GetClusterNameByServiceName(serviceName)

	//trigger cluster update
	cf.ca.TriggerClusterUpdate(clusterName, hosts)
}
