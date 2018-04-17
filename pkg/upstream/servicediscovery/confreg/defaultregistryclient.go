package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
)

type DefaultRegistryClient struct {
    systemConfig         *config.SystemConfig
    registryConfig       *config.RegistryConfig
    confregServerManager *servermanager.RegistryServerManager

    registerWorker   *registerWorker
    rpcServerManager servermanager.RPCServerManager

    stopConnChan chan bool
}

func NewRegistryClient(systemConfig *config.SystemConfig, registryConfig *config.RegistryConfig,
    manager *servermanager.RegistryServerManager) *DefaultRegistryClient {

    rc := &DefaultRegistryClient{
        systemConfig:         systemConfig,
        registryConfig:       registryConfig,
        confregServerManager: manager,
        stopConnChan:         make(chan bool),
    }

    rpcServerManager := servermanager.NewRPCServerManager()
    rc.rpcServerManager = rpcServerManager

    rw := NewRegisterWorker(systemConfig, registryConfig, manager, rpcServerManager)
    rc.registerWorker = rw

    return rc
}

func (rc *DefaultRegistryClient) GetRPCServerManager() servermanager.RPCServerManager {
    return rc.rpcServerManager
}

func (rc *DefaultRegistryClient) PublishAsync(dataId string, data ...string) {
    rc.registerWorker.SubmitPublishTask(dataId, data, model.EventTypePb_REGISTER.String())
}

func (rc *DefaultRegistryClient) UnPublishAsync(dataId string, data ...string) {
    rc.registerWorker.SubmitPublishTask(dataId, data, model.EventTypePb_UNREGISTER.String())
}

func (rc *DefaultRegistryClient) SubscribeAsync(dataId string) {
    rc.registerWorker.SubmitSubscribeTask(dataId, model.EventTypePb_REGISTER.String())
}

func (rc *DefaultRegistryClient) UnSubscribeAsync(dataId string) {
    rc.registerWorker.SubmitSubscribeTask(dataId, model.EventTypePb_UNREGISTER.String())
}

func (rc *DefaultRegistryClient) PublishSync(dataId string, data ...string) error {
    return rc.registerWorker.PublishSync(dataId, data)
}

func (rc *DefaultRegistryClient) UnPublishSync(dataId string, data ...string) error {
    return rc.registerWorker.UnPublishSync(dataId, data)
}

func (rc *DefaultRegistryClient) SubscribeSync(dataId string) error {
    return rc.registerWorker.SubscribeSync(dataId)
}

func (rc *DefaultRegistryClient) UnSubscribeSync(dataId string) error {
    return rc.registerWorker.UnSubscribeSync(dataId)
}