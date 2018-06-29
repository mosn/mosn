package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"testing"
)

func Test_AddClusterAndDump(t *testing.T) {
	
	log.InitDefaultLogger("", log.INFO)
	
	Load("../../../mosn/resource/mosn_config.json")
	
	//add cluster
	AddClusterConfig([]v2.Cluster{
		v2.Cluster{
			Name:           "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0@DEFAULT",
			ClusterType:    v2.DYNAMIC_CLUSTER,
			LbType:         v2.LB_RANDOM,
			Spec: v2.ClusterSpecInfo{
				Subscribes: []v2.SubscribeSpec{
					v2.SubscribeSpec{
						ServiceName: "x_test_service",
					},
				},
			},
		},
	})
	
	//change dump path, just for test
	ConfigPath = "../../../mosn/resource/mosn_config_dump_result.json"
	
	Dump(true)
}

func Test_RemoveClusterAndDump(t *testing.T) {
	
	log.InitDefaultLogger("", log.INFO)
	
}

func Test_ServiceRegistryInfoDump(t *testing.T) {
	
	log.InitDefaultLogger("", log.INFO)
	
}