package registry

import "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"

const CONFREG_SOFA_GROUP  = "SOFA"


type RegistryClient interface {
    Publish(dataId string, data ...string)

    UnPublish(dataId string, data ...string)

    Subscribe(dataId string)

    UnSubscribe(dataId string)

    GetRPCServerManager() servermanager.RPCServerManager
}
