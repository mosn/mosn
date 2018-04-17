package registry

import (
    "testing"
    _ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
    _ "gitlab.alipay-inc.com/afe/mosn/pkg/stream/sofarpc"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "fmt"
)

func Test_Publish(t *testing.T) {
    log.InitDefaultLogger("", log.INFO)

    var sysConfig = &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "confreg.sit.alipay.net",
        //RegistryEndpoint: "11.239.90.33",
        InstanceId: "000001",
        AppName:    "someApp",
    }

    var registryConfig = &config.RegistryConfig{
        ScheduleRegisterTaskDuration: 60 * 1000 * 1000 * 1000,
    }
    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    rc.Publish(someDataId, "dfsfds")
}

func Test_Register(t *testing.T) {
    log.InitDefaultLogger("", log.INFO)

    runRpcServer()

    var sysConfig = &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "confreg.sit.alipay.net",
        //RegistryEndpoint: "11.239.90.33",
        InstanceId: "000001",
        AppName:    "someApp",
    }

    var registryConfig = &config.RegistryConfig{
        ScheduleRegisterTaskDuration: time.Duration(5 * time.Second),
        RegisterTimeout:              time.Duration(3 * time.Second),
    }
    csm := servermanager.NewRegistryServerManager(sysConfig)

    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    anotherDataId := "anotherDataId"
    rc.UnPublish(someDataId, "dfsfds")
    //rc.Publish("anotherDataId", "dfsfds")

    rc.Subscribe(someDataId)
    rc.Subscribe(anotherDataId)
    //
    //rc.GetRPCServerManager().RegisterRPCServerChangeListener(&MockRPCServerChangeListener{})
    for ; ; {
        time.Sleep(5 * time.Second)
        rc.Publish(someDataId, "dfsfds")
    }
}

func Test_Received(t *testing.T) {
    log.InitDefaultLogger("", log.INFO)

    runRpcServer()

    var sysConfig = &config.SystemConfig{
        Zone:             "GZ00b",
        RegistryEndpoint: "confreg.sit.alipay.net",
        //RegistryEndpoint: "11.239.90.33",
        InstanceId: "000001",
        AppName:    "someApp",
    }

    var registryConfig = &config.RegistryConfig{
        ScheduleRegisterTaskDuration: time.Duration(5 * time.Second),
        RegisterTimeout:              time.Duration(3 * time.Second),
    }
    csm := servermanager.NewRegistryServerManager(sysConfig)

    rc := NewRegistryClient(sysConfig, registryConfig, csm)

    someDataId := "someDataId"
    //anotherDataId := "anotherDataId"

    rc.Subscribe(someDataId)
    //rc.Subscribe(anotherDataId)
    //
    rc.GetRPCServerManager().RegisterRPCServerChangeListener(&MockRPCServerChangeListener{})
    for ; ; {
        time.Sleep(5 * time.Second)
    }
}

func TestNewRegistryClient(t *testing.T) {
    c := make(chan bool)
    go func() {
        for ; ; {
            t := time.NewTimer(5 * time.Second)
            <-t.C
            c <- true
            fmt.Println("yyyy")
        }
    }()
    go func() {
        for ; ; {
            select {
            case <-c:
                fmt.Println("xx")
            }

        }
    }()

    for ; ; {
        time.Sleep(10 * time.Minute)
    }

}

type MockRPCServerChangeListener struct {
}

func (l *MockRPCServerChangeListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    fmt.Println("Changed: dataId = " + dataId)
    fmt.Println(zoneServers)
}
