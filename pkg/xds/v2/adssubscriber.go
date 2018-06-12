package v2

import "time"
import "gitlab.alipay-inc.com/afe/mosn/pkg/log"

func (adsClient *ADSClient) Start(v2Client *V2Client){
	adsClient.streamClient = adsClient.adsConfig.GetStreamClient()
	adsClient.v2Client = v2Client
	go adsClient.SendThread()
	go adsClient.ReceiveThread()
	time.Sleep(time.Second * 300)
}

func (adsClient *ADSClient) SendThread(){
	for {
		(*adsClient.v2Client).ReqListeners(adsClient.streamClient)
		time.Sleep(time.Second * 3)
	}
}

func (adsClient *ADSClient) ReceiveThread(){
	for {
		resp,err := adsClient.streamClient.Recv()
		if err != nil{
			log.DefaultLogger.Fatalf("get resp fail: %v", err)
		}
		typeUrl := resp.TypeUrl
		if typeUrl == "type.googleapis.com/envoy.api.v2.Listener"{
			log.DefaultLogger.Infof("get lds resp,handle it")
			(*adsClient.v2Client).HandleListersResp(resp)
		}else if typeUrl == "type.googleapis.com/envoy.api.v2.Cluster"{
			log.DefaultLogger.Infof("get cds resp,ignore now")
		}else if typeUrl == "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"{
			log.DefaultLogger.Infof("get eds resp,ignore now ")
		}
	}
}

func (adsClient *ADSClient) Stop(){

}