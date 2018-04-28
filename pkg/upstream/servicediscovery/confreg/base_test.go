package registry

import (
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "os"
)

var sysConfig *config.SystemConfig

func beforeTest() {
    log.InitDefaultLogger("", log.INFO)

    MockRpcServer()

    sysConfig = &config.SystemConfig{
        Zone:             "GZ00A",
        RegistryEndpoint: "http://confregsession-ci-04.inc.alipay.net",
        InstanceId:       "000001",
        AppName:          "someApp",
    }
}

func blockThread() {
    if os.Getenv("run_mode") == "test" {
        for ; ; {
            time.Sleep(5 * time.Second)
        }
    }
}
