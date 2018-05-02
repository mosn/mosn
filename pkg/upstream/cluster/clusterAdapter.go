package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
	"strings"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//added by  @boqin to register dynamic upstream update callback
func init() {
	cfa := &confregAdaptor{
		ca: &ClusterAdap,
	}
	RegisterUpstreamUpdateMethod(v2.DYNAMIC_CONFREG_CLUSTER, cfa)
}

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clustermnger *clusterManager
}

var adapterMap = make(map[v2.ClusterType]types.RegisterUpstreamUpdateMethodCb,4)

func RegisterUpstreamUpdateMethod(dtype v2.ClusterType, f types.RegisterUpstreamUpdateMethodCb) {
	adapterMap[dtype] = f
}

func (ca *ClusterAdapter) CallUpstreamUpdateMethod(providerType v2.ClusterType) {
	if v, ok := adapterMap[providerType]; ok {
		v.RegisterUpdateMethod()
	} else {
		log.DefaultLogger.Debugf("Type %s doesn't exist",string(providerType))
	}
}

type confregAdaptor struct {
	ca *ClusterAdapter
}

func (cf *confregAdaptor) RegisterUpdateMethod() {
	log.DefaultLogger.Debugf("[RegisterConfregListenerCb Called!]")
	servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(cf)
}

func (cf *confregAdaptor) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
	//11.166.22.163:12200?_TIMEOUT=3000&p=1&_SERIALIZETYPE=protobuf&_WARMUPTIME=0
	// &_WARMUPWEIGHT=10&app_name=bar1&zone=GZ00A&_MAXREADIDLETIME=30&_IDLETIMEOUT=27&v=4.0
	// &_WEIGHT=100&startTime=1524565802559
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
	//todo: update route according to services
	go func() {
		cf.ca.clustermnger.UpdateClusterHosts("confreg_service1", 0, hosts)
	}()
}
