package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "sync"
)

var confregServerManager *servermanager.RegistryServerManager
var registryClient RegistryClient

var lock = new(sync.Mutex)

var Initialization = false

func StartupRegistryModule(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig) RegistryClient {
    lock.Lock()

    defer func() {
        lock.Unlock()
    }()

    if Initialization {
        return registryClient
    }
    Initialization = true

    confregServerManager = servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    return NewRegistryClient(sysConfig, registryConfig, confregServerManager)
}
