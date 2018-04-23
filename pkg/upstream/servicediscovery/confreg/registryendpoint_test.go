package registry

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
)


func Test_RegistryModule(t *testing.T) {
    beforeTest()

    registryEndpoint := NewRegistryEndpoint(config.DefaultRegistryConfig, registryClient)

    registryEndpoint.StartListener()
}

