package v2

import "time"

import(
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
    envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)


func (adsClient *ADSClient) Start(){
	adsClient.StreamClient = adsClient.AdsConfig.GetStreamClient()
	adsClient.MosnConfig = &config.MOSNConfig{}
	go adsClient.SendThread()
	go adsClient.ReceiveThread()
}

func (adsClient *ADSClient) SendThread(){
	refreshDelay := adsClient.AdsConfig.RefreshDelay
	t1 := time.NewTimer(*refreshDelay)
	for {
		select {
		case <- adsClient.SendControlChan:
			log.DefaultLogger.Infof("send thread receive graceful shut down signal")
			adsClient.AdsConfig.CloseADSStreamClient()
			adsClient.StopChan <- 1
			return
		case <- t1.C:
			log.DefaultLogger.Infof("send thread request lds")
			err := adsClient.V2Client.ReqListeners(adsClient.StreamClient)
			if err != nil{
				log.DefaultLogger.Warnf("send thread request lds fail!auto retry next period")
			}
			log.DefaultLogger.Infof("send thread request cds")
			err = adsClient.V2Client.ReqClusters(adsClient.StreamClient)
			if err != nil{
				log.DefaultLogger.Warnf("send thread request cds fail!auto retry next period")
			}
			t1.Reset(*refreshDelay)
		}
	}
}

func (adsClient *ADSClient) ReceiveThread(){
	for {
		select {
		case <- adsClient.RecvControlChan:
			log.DefaultLogger.Infof("receive thread receive graceful shut down signal")
			adsClient.StopChan <- 2
			return
		default:
			resp,err := adsClient.StreamClient.Recv()
			if err != nil{
				log.DefaultLogger.Warnf("get resp timeout: %v", err)
				continue
			}
			typeUrl := resp.TypeUrl
			if typeUrl == "type.googleapis.com/envoy.api.v2.Listener"{
				log.DefaultLogger.Infof("get lds resp,handle it")
				listeners := adsClient.V2Client.HandleListersResp(resp)
				log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
				err := adsClient.MosnConfig.OnUpdateListeners(listeners)
				if err != nil {
					log.DefaultLogger.Fatalf("fail to update listeners")
					return
				}
				log.DefaultLogger.Infof("update listeners success")
			}else if typeUrl == "type.googleapis.com/envoy.api.v2.Cluster"{
				log.DefaultLogger.Infof("get cds resp,handle it")
				clusters := adsClient.V2Client.HandleClustersResp(resp)
				log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
				err := adsClient.MosnConfig.OnUpdateClusters(clusters)
				if err != nil {
					log.DefaultLogger.Fatalf("fall to update clusters")
					return
				}
				log.DefaultLogger.Infof("update clusters success")
				clusterNames := make([]string,0)
				for _,cluster := range clusters{
					if cluster.Type == envoy_api_v2.Cluster_EDS {
						clusterNames = append(clusterNames, cluster.Name)
					}
				}
				adsClient.V2Client.ReqEndpoints(adsClient.StreamClient, clusterNames)
			}else if typeUrl == "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"{
				log.DefaultLogger.Infof("get eds resp,handle it ")
				endpoints := adsClient.V2Client.HandleEndpointesResp(resp)
				log.DefaultLogger.Infof("get %d endpoints for cluster", len(endpoints))
				err = adsClient.MosnConfig.OnUpdateEndpoints(endpoints)
				if err != nil {
					log.DefaultLogger.Fatalf("fail to update endpoints for cluster")
					return
				}
				log.DefaultLogger.Infof("update endpoints for cluster %s success")
			}
		}
	}
}

func (adsClient *ADSClient) Stop(){
	adsClient.SendControlChan <- 1
	adsClient.RecvControlChan <- 1
	for i:=0;i<2;i++{
		select {
		case <- adsClient.StopChan:
			log.DefaultLogger.Infof("stop signal")
		}
	}
	close(adsClient.SendControlChan)
	close(adsClient.RecvControlChan)
	close(adsClient.StopChan)
}