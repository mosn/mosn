package registry

import (
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var sysConfig *config.SystemConfig
var registryConfig *config.RegistryConfig

func beforeTest() {
    log.InitDefaultLogger("", log.INFO)
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

func blockThread() {
    for ; ; {
        time.Sleep(5 * time.Second)
    }
}
