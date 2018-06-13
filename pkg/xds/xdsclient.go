package xds

import (
	"time"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds/v2"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
)

var warmuped chan bool = make(chan bool, 10)
var stopping chan bool = make(chan bool, 10)
var stopped chan bool = make(chan bool, 10)

type xdsClient struct {
	v2 *v2.V2Client
}

func (c *xdsClient) getConfig(config *config.MOSNConfig) (err error){

	log.DefaultLogger.Infof("start to get config from istio")
	chans := make([]chan bool, 2)
	chans[0] = make(chan bool)
	go c.getListenersAndRoutes(config, chans[0])
	chans[1] = make(chan bool)
	go c.getClustersAndHosts(config, chans[1])

	success := true
	for _, ch := range chans {
		v := <- ch
		success = success && v
		close(ch)
	}
	if !success {
		log.DefaultLogger.Fatalf("fail to get config from istio")
		err = errors.New("fail to get config from istio")
		return err
	}
	log.DefaultLogger.Infof("get config from istio success")
	return nil
}

func (c *xdsClient) getListenersAndRoutes(config *config.MOSNConfig, ch chan <- bool) {
	log.DefaultLogger.Infof("start to get listeners from LDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	listeners := c.v2.GetListeners(streamClient)
	if listeners == nil {
		log.DefaultLogger.Fatalf("get none listeners")
		ch <- false
		return
	}
	log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
	err := config.OnUpdateListeners(listeners)
	if err != nil {
		log.DefaultLogger.Fatalf("fail to update listeners")
		ch <- false
		return
	}
	log.DefaultLogger.Infof("update listeners success")
	ch <- true

}

func (c *xdsClient) getClustersAndHosts(config *config.MOSNConfig, ch chan <- bool) {
	log.DefaultLogger.Infof("start to get clusters from CDS")
	streamClient := c.v2.Config.ADSConfig.GetStreamClient()
	clusters := c.v2.GetClusters(streamClient)
	if clusters == nil {
		log.DefaultLogger.Fatalf("get none clusters")
		ch <- false
		return
	}
	log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
	err := config.OnUpdateClusters(clusters)
	if err != nil {
		log.DefaultLogger.Fatalf("fall to update clusters")
		ch <- false
		return
	}
	log.DefaultLogger.Infof("update clusters success")

	success := true
	for _, cluster := range clusters {
		//log.DefaultLogger.Infof("cluster: %+v \n", cluster)
		//fmt.Printf("cluster: %#v \n", cluster)
		if cluster.Type == xdsapi.Cluster_EDS {
			clusterName := cluster.Name
			log.DefaultLogger.Infof("start to get endpoints for cluster %s from EDS", clusterName)
			endpoints := c.v2.GetEndpoints(streamClient, clusterName)
			if endpoints == nil {
				log.DefaultLogger.Warnf("get none endpoints for cluster %s", clusterName)
				success = false
				continue
			}
			log.DefaultLogger.Infof("get %d endpoints for cluster %s", len(endpoints), clusterName)
			err = config.OnUpdateEndpoints(endpoints)
			if err != nil {
				log.DefaultLogger.Fatalf("fail to update endpoints for cluster %s", clusterName)
				success = false
				continue
			}
			log.DefaultLogger.Infof("update endpoints for cluster %s success", clusterName)
		}
	}
	ch <- success
}


func UnmarshalResources(config *config.MOSNConfig) (dynamicResources *bootstrap.Bootstrap_DynamicResources, staticResources *bootstrap.Bootstrap_StaticResources, err error) {
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



func Start(config *config.MOSNConfig, serviceCluster, serviceNode string) {

	log.DefaultLogger.Infof("xdsclient start\n")
	client := xdsClient{}
	if client.v2 == nil {
		dynamicResources, staticResources, err := UnmarshalResources(config)
		if err != nil {
			log.DefaultLogger.Fatalf("fail to unmarshal xds resources, skip xds: %v", err)
			warmuped <- true
			stopped <- true
			return
		}
		xdsConfig := v2.XDSConfig{}
		err = xdsConfig.Init(dynamicResources, staticResources)
		if err != nil {
			log.DefaultLogger.Fatalf("failt to init xds config, skip xds: %v", err)
			warmuped <- true
			stopped <- true
			return
		}
		client.v2 = &v2.V2Client{serviceCluster, serviceNode, &xdsConfig}
	}

	for {
		err := client.getConfig(config)
		if err == nil {
			break
		}
	}
	warmuped <- true

	refreshDelay := client.v2.Config.ADSConfig.RefreshDelay
	t1 := time.NewTimer(*refreshDelay)
	for {
		select {
		case <- stopping:
			client.v2.Config.ADSConfig.CloseADSStreamClient()
			stopped <- true
		case <- t1.C:
			client.getConfig(config)
			t1.Reset(*refreshDelay)
		}
	}

}

func StartV2(config *config.MOSNConfig, serviceCluster, serviceNode string) {
	log.DefaultLogger.Infof("xdsclient start\n")
	client := xdsClient{}
	if client.v2 == nil {
		dynamicResources, staticResources, err := UnmarshalResources(config)
		if err != nil {
			log.DefaultLogger.Fatalf("fail to unmarshal xds resources, skip xds: %v", err)
			warmuped <- true
			stopped <- true
			return
		}
		xdsConfig := v2.XDSConfig{}
		err = xdsConfig.Init(dynamicResources, staticResources)
		if err != nil {
			log.DefaultLogger.Fatalf("failt to init xds config, skip xds: %v", err)
			warmuped <- true
			stopped <- true
			return
		}
		client.v2 = &v2.V2Client{serviceCluster, serviceNode, &xdsConfig}
	}
	stopChan := make(chan int)
	sendControlChan := make(chan int)
	recvControlChan := make(chan int)
	adsClient := &v2.ADSClient{
		AdsConfig: client.v2.Config.ADSConfig,
		StreamClient: nil,
		V2Client: client.v2,
		MosnConfig: nil,
		SendControlChan: sendControlChan,
		RecvControlChan: recvControlChan,
		StopChan: stopChan,
	}
	adsClient.Start()
	time.Sleep(time.Second * 30)
	adsClient.Stop()
}

func Stop() {
	log.DefaultLogger.Infof("prepare to stop xdsclient\n")
	stopping <- true
	<- stopped
	log.DefaultLogger.Infof("xdsclient stop\n")
}

func WaitForWarmUp() {
	<- warmuped
}
