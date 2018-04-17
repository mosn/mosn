package registry

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "time"
)

var sysConfig *config.SystemConfig
var registryConfig *config.RegistryConfig

func TestMsgChannel_PublishService(t *testing.T) {
    before()

    StartupRegistryModule(sysConfig, registryConfig)

    registryClient, _ := GetRegistryClient()
    msgChann := NewMsgChannel(nil, nil, registryClient)

    msgChann.StartChannel()
}

func before() {
    MockRpcServer()
    
    sysConfig = &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "confreg.sit.alipay.net",
        InstanceId:       "000001",
        AppName:          "someApp",
    }
    registryConfig = &config.RegistryConfig{
        ScheduleRegisterTaskDuration: 60 * time.Second,
        RegisterTimeout:              3 * time.Second,
    }
}
