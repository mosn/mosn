package registry

import (
    "testing"
)


func Test_RegistryModule(t *testing.T) {
    beforeTest()

    registryEndpoint := NewRegistryEndpoint(registryConfig, registryClient)

    registryEndpoint.StartListener()
}

