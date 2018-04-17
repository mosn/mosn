package servermanager

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "fmt"
    "time"
)

func Test(t *testing.T) {
    rpcSvrManager := NewRPCServerManager()

    rpcSvrManager.RegisterRPCServerChangeListener(&MyRPCServerChangeListener{})

    rpcSvrManager.RegisterRPCServer(assembleReceivedData())
    rpcSvrManager.RegisterRPCServer(assembleReceivedData2())
    rpcSvrManager.RegisterRPCServer(assembleReceivedData3())

    dataId := "someDataId1"
    srvs := rpcSvrManager.GetRPCServerList(dataId)
    fmt.Println(srvs)
    zone2Srvs := rpcSvrManager.GetRPCServerListByZone(dataId, "zone2")
    zone3Srvs := rpcSvrManager.GetRPCServerListByZone(dataId, "zone3")
    fmt.Println(zone2Srvs)
    fmt.Println(zone3Srvs)

    for ; ; {
        time.Sleep(5 * time.Second)

    }
}

func TestCoverd(t *testing.T) {
    rpcSvrManager := NewRPCServerManager()

    rpcSvrManager.RegisterRPCServerChangeListener(&MyRPCServerChangeListener{})
    //first time
    rpcSvrManager.RegisterRPCServer(assembleReceivedData())
    rpcSvrManager.RegisterRPCServer(assembleReceivedData4())

    for ; ; {
        time.Sleep(5 * time.Second)

    }
}

type MyRPCServerChangeListener struct {
}

func (l *MyRPCServerChangeListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    fmt.Println("Changed")
    fmt.Println(zoneServers)
}

func assembleReceivedData() *model.ReceivedDataPb {
    dataBox := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"data1"}, {"data2"}, {"data3"}},
    }

    dataBox2 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"c1"}, {"c2"}, {"c3"}},
    }

    rd := &model.ReceivedDataPb{
        DataId:  "someDataId1",
        Segment: "s1",
        Data:    map[string]*model.DataBoxesPb{"zone1": dataBox, "zone2": dataBox2},
        Version: 1,
    }

    return rd
}

func assembleReceivedData2() *model.ReceivedDataPb {
    dataBox3 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"t1"}, {"t2"}, {"t3"}},
    }

    dataBox4 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"v1"}, {"v2"}, {"v3"}},
    }

    rd := &model.ReceivedDataPb{
        DataId:  "someDataId1",
        Segment: "s2",
        Data:    map[string]*model.DataBoxesPb{"zone3": dataBox3, "zone4": dataBox4},
    }

    return rd
}

func assembleReceivedData3() *model.ReceivedDataPb {
    dataBox := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"a1"}, {"a2"}, {"a3"}},
    }

    dataBox2 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"d1"}, {"d2"}, {"d4"}},
    }

    rd := &model.ReceivedDataPb{
        DataId:  "someDataId2",
        Segment: "s1",
        Data:    map[string]*model.DataBoxesPb{"zone1": dataBox, "zone2": dataBox2},
    }

    return rd

}

func assembleReceivedData4() *model.ReceivedDataPb {
    dataBox := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"a1"}, {"a2"}, {"a3"}},
    }

    dataBox2 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"d1"}, {"d2"}, {"d4"}},
    }

    rd := &model.ReceivedDataPb{
        DataId:  "someDataId1",
        Segment: "s1",
        Data:    map[string]*model.DataBoxesPb{"zone1": dataBox, "zone2": dataBox2},
        Version: 1,
    }

    return rd

}
