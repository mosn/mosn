package registry

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "fmt"
    "time"
)

func Test_AsyncPublishOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")

    blockThread()
}

func Test_SyncPublishOneService(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    err := rc.PublishSync(someDataId, "a", "b", "c")
    if err != nil {
        t.Fatal(err)
    } else {
        fmt.Println("Publish success.")
    }
}

func Test_AsyncPublishOneServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    for ; ; {
        <-time.After(1 * time.Second)
        someDataId := "someDataId"
        rc.PublishAsync(someDataId, "PublishData-"+RandomUuid())
    }
}

func Test_SyncPublishOneServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    for i := 0; i < 10; i++ {
        go func() {
            <-time.After(2 * time.Second)
            rc.PublishSync(someDataId, RandomUuid(), RandomUuid(), RandomUuid())
        }()
    }
    blockThread()
}

func Test_AsyncPublishManyServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    for i := 0; i < 3; i++ {
        someDataId := "ServiceName-" + RandomUuid()
        go func() {
            for ; ; {
                <-time.After(3 * time.Second)
                rc.PublishAsync(someDataId, "PublishData-"+RandomUuid())
            }
        }()
    }

    blockThread()
}

func Test_SyncPublishManyServiceManyTimes(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    for i := 0; i < 10; i++ {
        someDataId := "ServiceName-" + RandomUuid()
        go func() {
            for ; ; {
                <-time.After(3 * time.Second)
                rc.PublishSync(someDataId, "PublishData-"+RandomUuid())
            }
        }()
    }

    blockThread()
}

func Test_AsyncPublishTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")

    blockThread()
}

func Test_SyncPublishTimeout(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    err := rc.PublishSync(someDataId, "a", "b", "c")
    if err != nil {
        fmt.Println("Publish sync failed. err = ", err)
    }

    blockThread()
}

func Test_AsyncPublishThreeServiceAndOneFailed(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    rc := NewConfregClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.PublishAsync(someDataId, "a", "b", "c")

    anotherDataId := "anotherDataId"
    rc.PublishAsync(anotherDataId, "a", "b", "c")

    thirdDataId := "thirdDataId"
    rc.PublishAsync(thirdDataId, "a", "b", "c")

    blockThread()
}

