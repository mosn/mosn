package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "sync"
    "errors"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var confregServerManager *servermanager.RegistryServerManager
var registryClient RegistryClient

var lock = new(sync.Mutex)

var Initialization = false

func StartupRegistryModule(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig) {
    lock.Lock()

    defer func() {
        lock.Unlock()
    }()

    if Initialization {
        return
    }
    Initialization = true
    log.InitDefaultLogger("", log.INFO)

    confregServerManager = servermanager.NewRegistryServerManager(sysConfig)
    registryClient = NewRegistryClient(sysConfig, registryConfig, confregServerManager)

}

func GetRegistryClient() (RegistryClient, error) {
    if !Initialization {
        return nil, errors.New("registry client is not Initialization")
    }

    return registryClient, nil
}
