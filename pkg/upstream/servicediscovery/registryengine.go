package servicediscovery

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "time"
)

var inited bool = false
var sysConfig config.SystemConfig

func init() {
    if inited {
        return
    }
    inited = true


}

func initSystemConfig()  {
}

func initRegistryConfig() *config.RegistryConfig {
    return &config.RegistryConfig{
        ScheduleRegisterTaskDuration: time.Duration(1000),
    }
}
