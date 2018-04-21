package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "testing"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
)

func Test_AsyncPubSubOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")
    rc.SubscribeAsync("subDataId")

    blockThread()
}

func Test_yncPubSubOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishSync(someDataId, "a", "b", "c")
    rc.SubscribeSync("subDataId")

    blockThread()
}

func Test_AsyncPubSubOneServiceAndPubTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")
    rc.SubscribeAsync("subDataId")

    blockThread()
}

func Test_SyncPubSubOneServiceAndPubTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishSync(someDataId, "a", "b", "c")
    rc.SubscribeSync("subDataId")

    blockThread()
}

func Test_AsyncPubSubOneServiceAndPubSubTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")
    rc.SubscribeAsync("subDataId")

    blockThread()
}

func Test_AsyncPubSubManyServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, config.DefaultRegistryConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    go func() {
        for i := 0; i < 3; i++ {
            pubDataId := "PubDataId-" + RandomUuid()
            for ; ; {
                time.Sleep(3 * time.Second)
                rc.PublishAsync(pubDataId, RandomUuid(), RandomUuid())
            }
        }
    }()
    go func() {
        for i := 0; i < 3; i++ {
            subDataId := "SubDataId-" + RandomUuid()
            for ; ; {
                time.Sleep(3 * time.Second)
                rc.SubscribeAsync(subDataId)
            }
        }
    }()

    blockThread()
}
