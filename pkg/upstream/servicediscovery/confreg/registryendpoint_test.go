package registry

import (
    "testing"
)


func Test_RegistryModule(t *testing.T) {
    beforeTest()

    registryEndpoint := NewRegistryEndpoint(nil, nil, registryClient)

    registryEndpoint.StartListener()
}

