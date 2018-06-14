package xds

import (
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v2"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
	//"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//var warmuped chan bool = make(chan bool)
//var stopping chan bool = make(chan bool)
//var stopped chan bool = make(chan bool)
//var started bool = false

type XdsClient struct {
	v2 *v2.V2Client
	adsClient *v2.ADSClient
}

func (c *XdsClient) getConfig(config *config.MOSNConfig) error{
	log.DefaultLogger.Infof("start to get config from istio")
	err := c.getListenersAndRoutes(config)
	if err != nil {
		log.DefaultLogger.Fatalf("fail to get lds config from istio")
		return err
	}
	err = c.getClustersAndHosts(config)
	if err != nil {
		log.DefaultLogger.Fatalf("fail to get cds config from istio")
		return err
	}
	log.DefaultLogger.Infof("get config from istio success")
	return nil
}

func (c *XdsClient) getListenersAndRoutes(config *config.MOSNConfig) error{
	log.DefaultLogger.Infof("start to get listeners from LDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	listeners := c.v2.GetListeners(streamClient)
	if listeners == nil {
		log.DefaultLogger.Fatalf("get none listeners")
		return errors.New("get none listener")
	}
	log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	err := config.OnUpdateListeners(listeners)
	if err != nil {
		log.DefaultLogger.Fatalf("fail to update listeners")
		return errors.New("fail to update listeners")
	}
	log.DefaultLogger.Infof("update listeners success")
	return nil
}

func (c *XdsClient) getClustersAndHosts(config *config.MOSNConfig) error{
	log.DefaultLogger.Infof("start to get clusters from CDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	clusters := c.v2.GetClusters(streamClient)
	if clusters == nil {
		log.DefaultLogger.Fatalf("get none clusters")
		return errors.New("get none clusters")
	}
	log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	err := config.OnUpdateClusters(clusters)
	if err != nil {
		log.DefaultLogger.Fatalf("fall to update clusters")
		return errors.New("fail to update clusters")
	}
	log.DefaultLogger.Infof("update clusters success")

	clusterNames := make([]string,0)
	for _, cluster := range clusters {
		if cluster.Type == xdsapi.Cluster_EDS {
			clusterNames = append(clusterNames, cluster.Name)
		}
	}
	log.DefaultLogger.Infof("start to get endpoints for cluster %v from EDS", clusterNames)
	endpoints := c.v2.GetEndpoints(streamClient, clusterNames)
	if endpoints == nil {
		log.DefaultLogger.Warnf("get none endpoints for cluster %v", clusterNames)
		return errors.New("get none endpoints for clusters")
	}
	log.DefaultLogger.Infof("get %d endpoints for cluster %v", len(endpoints), clusterNames)
	err = config.OnUpdateEndpoints(endpoints)
	if err != nil {
		log.DefaultLogger.Fatalf("fail to update endpoints for cluster %v", clusterNames)
		return errors.New("fail to update endpoints for clusters")
	}
	log.DefaultLogger.Infof("update endpoints for cluster %v success", clusterNames)
	return nil
}


func (c *XdsClient)UnmarshalResources(config *config.MOSNConfig) (dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources, err error) {
	if len(config.RawDynamicResources) > 0 {
		dynamicResources = &bootstrap.Bootstrap_DynamicResources{}
		err = jsonpb.UnmarshalString(string(config.RawDynamicResources), dynamicResources)
		//if err != nil {
		//	return nil, nil, err
		//}
		err = dynamicResources.Validate()
		if err != nil {
			return nil, nil, err
		}
	}
	if len(config.RawStaticResources) > 0 {
		staticResources = &bootstrap.Bootstrap_StaticResources{}
		err = jsonpb.UnmarshalString(string(config.RawStaticResources), staticResources)
		//if err != nil {
		//	return nil,nil,err
		//}
		err = staticResources.Validate()
		if err != nil {
			return nil,nil,err
		}
	}
	return dynamicResources, staticResources, nil
}



//func Start(config *config.MOSNConfig, serviceCluster, serviceNode string) {
//
//	log.DefaultLogger.Infof("xdsclient start\n")
//	client := XdsClient{}
//	if client.v2 == nil {
//		dynamicResources, staticResources, err := UnmarshalResources(config)
//		if err != nil {
//			log.DefaultLogger.Fatalf("fail to unmarshal xds resources, skip xds: %v", err)
//			warmuped <- false
//			return
//		}
//		xdsConfig := v2.XDSConfig{}
//		err = xdsConfig.Init(dynamicResources, staticResources)
//		if err != nil {
//			log.DefaultLogger.Fatalf("failt to init xds config, skip xds: %v", err)
//			warmuped <- false
//			return
//		}
//		client.v2 = &v2.V2Client{serviceCluster, serviceNode, &xdsConfig}
//	}
//
//	for {
//		err := client.getConfig(config)
//		if err == nil {
//			break
//		}
//	}
//	warmuped <- true
//	started = true
//
//	refreshDelay := client.v2.Config.ADSConfig.RefreshDelay
//	t1 := time.NewTimer(*refreshDelay)
//	for {
//		select {
//		case <- stopping:
//			client.v2.Config.ADSConfig.CloseADSStreamClient()
//			stopped <- true
//		case <- t1.C:
//			client.getConfig(config)
//			t1.Reset(*refreshDelay)
//		}
//	}
//
//}

func (c *XdsClient)Start(config *config.MOSNConfig, serviceCluster, serviceNode string) error{
	log.DefaultLogger.Infof("xds client start")
	if c.v2 == nil {
		dynamicResources, staticResources, err := c.UnmarshalResources(config)
		if err != nil {
			log.DefaultLogger.Fatalf("fail to unmarshal xds resources, skip xds: %v", err)
			return errors.New("fail to unmarshal xds resources")
		}
		xdsConfig := v2.XDSConfig{}
		err = xdsConfig.Init(dynamicResources, staticResources)
		if err != nil {
			log.DefaultLogger.Fatalf("fail to init xds config, skip xds: %v", err)
			return errors.New("fail to init xds config")
		}
		c.v2 = &v2.V2Client{serviceCluster, serviceNode, &xdsConfig}
	}
	err := c.getConfig(config)
	if err != nil {
		return err
	}
	stopChan := make(chan int)
	sendControlChan := make(chan int)
	recvControlChan := make(chan int)
	adsClient := &v2.ADSClient{
		AdsConfig: c.v2.Config.ADSConfig,
		StreamClient: nil,
		V2Client: c.v2,
		MosnConfig: nil,
		SendControlChan: sendControlChan,
		RecvControlChan: recvControlChan,
		StopChan: stopChan,
	}
	adsClient.Start()
	c.adsClient = adsClient
	return nil
}

func (c *XdsClient)Stop() {
	log.DefaultLogger.Infof("prepare to stop xds client")
	c.adsClient.Stop()
	log.DefaultLogger.Infof("xds client stop")
}

// must be call after func start
//func (c *XdsClient)WaitForWarmUp() {
//	<- warmuped
//}
