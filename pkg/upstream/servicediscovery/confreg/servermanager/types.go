package servermanager

import "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"

type RPCServerManager interface {
    RegisterRPCServer(receivedData *model.ReceivedDataPb)

    GetRPCServerList(dataId string) (servers map[string][]string, ok bool)

    GetRPCServerListWithoutZone(dataId string) (servers []string, ok bool)

    GetRPCServerListByZone(dataId string, zone string) (servers []string, ok bool)

    GetRPCServiceSnapshot() []byte

    RegisterRPCServerChangeListener(listener RPCServerChangeListener)
}

type RPCServerChangeListener interface {
    OnRPCServerChanged(dataId string, zoneServers map[string][]string)
}

type RegistryServerChangeListener interface {
    OnRegistryServerChangeEvent(registryServers []string)
}
