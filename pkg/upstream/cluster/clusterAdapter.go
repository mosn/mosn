package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
	"strings"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//added by  @boqin：update cluster dynamically,update upstream dynamically, update route dynamically
func init() {
	cfa := &confregAdaptor{
		ca: &ClusterAdap,
	}
	SetAdapterMap(v2.CONFREG_CLUSTER, cfa)
}

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng             *clusterManager
}

var adapterMap = make(map[v2.SubClusterType]types.RegisterUpstreamUpdateMethodCb, 4)

func SetAdapterMap(sct v2.SubClusterType, f types.RegisterUpstreamUpdateMethodCb) {
	adapterMap[sct] = f
}

func (ca *ClusterAdapter) DoRegister(providerType v2.SubClusterType) {
	if v, ok := adapterMap[providerType]; ok {
		v.RegisterUpdateMethod()
	} else {
		log.DefaultLogger.Debugf("Type %s doesn't exist", string(providerType))
	}
}

type confregAdaptor struct {
	ca    *ClusterAdapter
	isReg bool
}

//todo: confreg module starting here according to config file
func (cf *confregAdaptor) RegisterUpdateMethod() {
	log.DefaultLogger.Debugf("[RegisterConfregListenerCb Called!]")
	if !cf.isReg {
		servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(cf)
		cf.isReg = true
	}
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
	cf.TriggerClusterUpdate(clusterName, hosts)
}

func (cf *confregAdaptor) TriggerClusterUpdate(clusterName string, hosts []v2.Host) {

	//update cluster
	clusterExist := cf.ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {

		cluster := v2.Cluster {
			Name:           clusterName,
			ClusterType:    v2.DYNAMIC_CLUSTER,
			SubClustetType: v2.CONFREG_CLUSTER,
			LbType:         v2.LB_RANDOM,
		}
		//new cluster
		cf.ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
	}
	//add hosts to cluster
	cf.ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}