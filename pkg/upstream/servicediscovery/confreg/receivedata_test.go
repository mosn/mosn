package registry

import (
    "testing"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "fmt"
)

type TestRPCServerChangeListener struct {
}

func (t *TestRPCServerChangeListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    receivedDataChan <- true
}

var listener = &TestRPCServerChangeListener{}
var receivedDataChan = make(chan bool)

func Test_OpenCC(t *testing.T) {
    MockRpcServer()
    blockThread()
}

func Test_ReceiveSingleSegmentData(t *testing.T) {
    beforeTest()

    csm := servermanager.NewRegistryServerManager(sysConfig)
    rc := NewRegistryClient(sysConfig, registryConfig, csm)
    rc.GetRPCServerManager().RegisterRPCServerChangeListener(listener)

    re := NewRegistryEndpoint(nil, nil, rc)
    go re.StartListener()
    
    dataId := "someDataId"
    rc.SubscribeSync(dataId)

    for ; ; {
        select {
        case <-receivedDataChan:
            {
                data, ok := rc.GetRPCServerManager().GetRPCServerListByZone(dataId, "zone2")
                if !ok {
                    fmt.Println("empty server list for data id = ", dataId)
                } else {
                    fmt.Println("Server List = ", data)
                }

            }
        }
    }

    blockThread()
}
