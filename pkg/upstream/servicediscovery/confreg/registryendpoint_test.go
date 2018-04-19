package registry

import (
    "testing"
)


func TestMsgChannel_PublishService(t *testing.T) {
    beforeTest()

    StartupRegistryModule(sysConfig, registryConfig)

    registryClient, _ := GetRegistryClient()
    msgChann := NewRegistryEndpoint(nil, nil, registryClient)

    msgChann.StartListener()
}

