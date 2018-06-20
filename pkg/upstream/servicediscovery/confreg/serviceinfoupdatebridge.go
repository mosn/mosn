package registry

import (
	"errors"
	"strings"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

type confregAdaptor struct {
	ca *cluster.ClusterAdapter
}

func init() {
	config.RegisterConfigParsedListener(config.ParseCallbackKeyCluster, OnClusterInfoParsed)
	config.RegisterConfigParsedListener(config.ParseCallbackKeyServiceRgtInfo, OnServiceRegistryInfoParsed)
}

func (cf *confregAdaptor) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
	log.StartLogger.Debugf("Call back by confreg,%+v", zoneServers)
	//dataId = dataId[:len(dataId)-8]
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
				log.StartLogger.Debugf("dataId is %s, IP_ADDR is %s", dataId, ipaddress)
			}
		}
	}

	// note: at this time, serviceName == clusterName
	cf.ca.TriggerClusterUpdate(serviceName, hosts)

	//router.RoutersManager.AddRouterInRouters([]string{dataId})
}

// del dynamic cluster from memory
func TriggerDynamicClusterClear() {
	log.StartLogger.Debugf("Call back by confreg to reset dynamic cluster")

	for _, serviceName := range subServiceList {
		cluster.ClusterAdap.TriggerClusterDel(serviceName)
	}
	//go router.RoutersManager.RemoveRouterInRouters(subServiceList)
}

var (
	subServiceList   []string
	pubServiceList   = make(map[string]string)
	registryAppInfo  = v2.ApplicationInfo{}
	configParseReady bool
)

// Called by registry module
// In turn: GetAppInfoFromConfig -> GetSubListFromConfig -> GetPubListFromConfig
func GetAppInfoFromConfig() v2.ApplicationInfo {
	return registryAppInfo
}

func GetSubListFromConfig() []string {
	return subServiceList
}

func GetPubListFromConfig() map[string]string {
	return pubServiceList
}

// Called when register receive http's application registry info
// Working when mesh has parsed and wrote back all registry info already for avoiding data corruption
func ResetRegistryInfo(appInfo v2.ApplicationInfo) {
	for {
		if configParseReady {
			TriggerDynamicClusterClear()
			ResetApplicationInfo(appInfo)
			subServiceList = []string{}
			pubServiceList = map[string]string{}
			return
		}
	}
}

func ResetApplicationInfo(appInfo v2.ApplicationInfo) bool {
	registryAppInfo = appInfo
	config.ResetServiceRegistryInfo(registryAppInfo, subServiceList)
	return true
}

func AddSubInfo(subInfo []string) bool {
	log.DefaultLogger.Debugf("AddSubInfo", subInfo)
	var clustersAdded []v2.Cluster

	for _, si := range subInfo {
		exist := false
		for _, s := range subServiceList {
			if si == s {
				exist = true
				break
			}
		}
		if exist {
			continue
		}

		subServiceList = append(subServiceList, si)
		v2Cluster := v2.Cluster{
			Name:              si,
			ClusterType:       v2.DYNAMIC_CLUSTER,
			SubClustetType:    v2.CONFREG_CLUSTER,
			LbType:            v2.LB_RANDOM,
			MaxRequestPerConn: 1024,
			ConnBufferLimitBytes: 32 * 1024,
			CirBreThresholds:  v2.CircuitBreakers{},
			Spec: v2.ClusterSpecInfo{
				Subscribes: []v2.SubscribeSpec{
					v2.SubscribeSpec{
						ServiceName: si,
					},
				},
			},
		}

		clustersAdded = append(clustersAdded, v2Cluster)
		cluster.ClusterAdap.TriggerClusterAdded(v2Cluster)
	}

	if len(clustersAdded) != 0 {
		config.AddClusterConfig(clustersAdded)
	}
	return true
}

func DelSubInfo(subinfo []string) bool {
	//go router.RoutersManager.RemoveRouterInRouters(subinfo)
	log.DefaultLogger.Debugf("DelSubInfo", subinfo)

	for _, sub := range subinfo {
		for i, s := range subServiceList {
			if sub == s {
				subServiceList = append(subServiceList[:i], subServiceList[i+1:]...)
				break
			}
		}
	}

	for _, serviceName := range subinfo {
		cluster.ClusterAdap.TriggerClusterDel(serviceName)
	}
	go config.RemoveClusterConfig(subinfo)
	return true
}

func AddPubInfo(pubInfo map[string]string) bool {
	config.AddPubInfo(pubInfo)
	return true
}

func DelPubInfo(serviceName string) bool {

	delete(pubServiceList, serviceName)
	config.DelPubInfo(serviceName)
	return true
}

// Callback by config parsing ready, in turn OnAppInfo->OnPubInfo->OnSubInfo
// To make sure use config's registry info first before writing http's push info
func OnAppInfoParsed(appInfo v2.ApplicationInfo) {
	registryAppInfo = appInfo
}

func OnSubInfoParsed(subInfo []string) {
	subServiceList = subInfo
}

// key is service_name
func OnPubInfoParsed(pubInfo map[string]string) {
	pubServiceList = pubInfo
}

// Called when cluster info parsed ready
func OnClusterInfoParsed(data interface{}, endParsed bool) error {
	subInfo := []string{}

	if clusters, ok := data.([]v2.Cluster); ok {
		for _, c := range clusters {
			for _, sub := range c.Spec.Subscribes {
				subInfo = append(subInfo, sub.ServiceName)
			}
		}
	} else {
		var err error = errors.New("invalid value passed")
		log.DefaultLogger.Fatalf("%+v invalid value passed", data)
		return err
	}

	OnSubInfoParsed(subInfo)

	if endParsed {
		configParseReady = true
		RecoverRegistryModule()
	}
	return nil
}

// Called when service registry parsed ready
// endParsed = true, when registry info parsed
func OnServiceRegistryInfoParsed(data interface{}, endParsed bool) error {
	var appInfoArray v2.ApplicationInfo
	var pubList = make(map[string]string)

	if serviceRegInfo, ok := data.(v2.ServiceRegistryInfo); ok {
		appInfoArray = serviceRegInfo.ServiceAppInfo

		for _, pubInfo := range serviceRegInfo.ServicePubInfo {
			pubList[pubInfo.Pub.ServiceName] = pubInfo.Pub.PubData
		}
	} else {
		var err error = errors.New("invalid value passed")
		log.DefaultLogger.Fatalf("%+v invalid value passed", data)
		return err
	}

	OnAppInfoParsed(appInfoArray)
	OnPubInfoParsed(pubList)

	if endParsed {
		configParseReady = true
		RecoverRegistryModule()
	}
	return nil
}
