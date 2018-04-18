package registry

import (
    "testing"
    _ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
    _ "gitlab.alipay-inc.com/afe/mosn/pkg/stream/sofarpc"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "fmt"
)


func Test_Publish(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "dfsfds")

    blockThread()
}

func Test_Register(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)

    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    anotherDataId := "anotherDataId"
    rc.SubscribeAsync(someDataId)
    rc.SubscribeAsync(anotherDataId)

    blockThread()
}

func Test_Received(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)

    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeAsync(someDataId)
    rc.GetRPCServerManager().RegisterRPCServerChangeListener(&MockRPCServerChangeListener{})

    blockThread()
}

type MockRPCServerChangeListener struct {
}

func (l *MockRPCServerChangeListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    fmt.Println("Changed: dataId = " + dataId)
    fmt.Println(zoneServers)
}
