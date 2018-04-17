package servermanager

import "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"

type RPCServerManager interface {

    RegisterRPCServer(receivedData *model.ReceivedDataPb)

    GetRPCServerList(dataId string) map[string][]string

    GetRPCServerListByZone(dataId string, zone string) []string

    RegisterRPCServerChangeListener(listener RPCServerChangeListener)
}

type RPCServerChangeListener interface {
    OnRPCServerChanged(dataId string, zoneServers map[string][]string)
}

type RegistryServerChangeListener interface {
    OnRegistryServerChangeEvent(registryServers []string)
}