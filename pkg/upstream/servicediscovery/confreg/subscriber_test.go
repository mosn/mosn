package registry

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "time"
)

func Test_AsyncSubscribeOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeAsync(someDataId)

    blockThread()
}

func Test_SyncSubscribeOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeSync(someDataId)

    blockThread()
}

func Test_AsyncSubscribeOneServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    for ; ; {
        <-time.After(1 * time.Second)
        rc.SubscribeAsync(someDataId)
    }

    blockThread()
}

func Test_SyncSubscribeOneServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    for ; ; {
        <-time.After(1 * time.Second)
        rc.SubscribeSync(someDataId)
    }

    blockThread()
}

func Test_AsyncSubscribeManyServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    for i := 0; i < 3; i++ {
        someDataId := "ServiceName-" + RandomUuid()
        go func() {
            for ; ; {
                <-time.After(1 * time.Second)
                rc.SubscribeAsync(someDataId)
            }
        }()
    }

    blockThread()
}

func Test_SyncSubscribeManyServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    for i := 0; i < 10; i++ {
        someDataId := "ServiceName-" + RandomUuid()
        go func() {
            for ; ; {
                <-time.After(3 * time.Second)
                rc.SubscribeSync(someDataId)
            }
        }()
    }

    blockThread()
}

func Test_AsyncSubscribeTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeAsync(someDataId)

    blockThread()
}

func Test_SyncSubscribeTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeSync(someDataId)

    blockThread()
}

func Test_AsyncSubscribeThreeServiceAndOneFailed(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.SubscribeAsync(someDataId)

    anotherDataId := "anotherDataId"
    rc.SubscribeAsync(anotherDataId)

    thirdDataId := "thirdDataId"
    rc.SubscribeAsync(thirdDataId)

    blockThread()
}