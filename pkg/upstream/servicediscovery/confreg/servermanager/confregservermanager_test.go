package servermanager

import (
    "testing"
    "fmt"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
)

func TestConfregServerManager_WrongEndpoint(t *testing.T) {
    beforeTest()

    sysConfig := &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "11.239.90.330",
    }

    NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)

}
func TestConfregServerManager_GetConfregServerByRandom(t *testing.T) {
    beforeTest()

    sysConfig := &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "11.239.90.33",
    }
    csm := NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)

    server, ok := csm.GetRegistryServerByRandom()
    if ok {
        fmt.Println(server)
    }

    servers, ok := csm.GetRegistryServerList()
    if ok {
        fmt.Println(servers)
    }

    blockThread()
}

func TestRegistryServerManager_IsChanged(t *testing.T) {
    old := []string{"a", "b", "c"}
    new := []string{"a", "d", "c"}
    new2 := []string{"a", "b", "c", "d"}
    new3 := []string{"a", "b", "c"}

    fmt.Println(isChanged(nil, new))
    fmt.Println(isChanged(old, new2))
    fmt.Println(isChanged(old, new3))
}
