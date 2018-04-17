package servermanager

import (
    "testing"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "fmt"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
)

func TestConfregServerManager_GetConfregServerByRandom(t *testing.T) {
    log.InitDefaultLogger("", log.INFO)
    sysConfig := &config.SystemConfig{
        Zone: "GZ00b",
        //RegistryEndpoint: "confreg.sit.alipay.net",
        RegistryEndpoint: "11.239.90.33",
    }
    csm := NewRegistryServerManager(sysConfig)

    server, err := csm.GetRegistryServerByRandom()
    if err != nil {
        fmt.Print(err)
    } else {
        fmt.Print(server)
    }

    for ; ; {
        fmt.Println(csm.GetRegistryServerByRandom())
        time.Sleep(2 * time.Second)
    }
}

func TestRegistryServerManager_IsChanged(t *testing.T) {
    old := []string{"a", "b", "c"}
    new := []string{"a", "d", "c"}
    new2 := []string{"a", "b", "c", "d"}
    new3 := []string{"a", "b", "c"}

    fmt.Println(isChanged(old, new))
    fmt.Println(isChanged(old, new2))
    fmt.Println(isChanged(old, new3))
}