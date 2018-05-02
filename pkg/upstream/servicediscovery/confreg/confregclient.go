package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
)

const ConfregSofaGroup = "SOFA"

type ConfregClient struct {
    systemConfig         *config.SystemConfig
    registryConfig       *config.RegistryConfig
    confregServerManager *servermanager.RegistryServerManager

    registerWorker   *registerWorker
    rpcServerManager servermanager.RPCServerManager

    stopConnChan chan bool
}

func NewConfregClient(systemConfig *config.SystemConfig, registryConfig *config.RegistryConfig,
    manager *servermanager.RegistryServerManager) Client {

    rc := &ConfregClient{
        systemConfig:         systemConfig,
        registryConfig:       registryConfig,
        confregServerManager: manager,
        stopConnChan:         make(chan bool),
    }

    rc.rpcServerManager = servermanager.GetRPCServerManager()

    rw := NewRegisterWorker(systemConfig, registryConfig, manager, rc.rpcServerManager)
    rc.registerWorker = rw

    return rc
}

func (rc *ConfregClient) GetRPCServerManager() servermanager.RPCServerManager {
    return rc.rpcServerManager
}

func (rc *ConfregClient) PublishAsync(dataId string, data ...string) {
    rc.registerWorker.SubmitPublishTask(dataId, data, model.EventTypePb_REGISTER.String())
}

func (rc *ConfregClient) UnPublishAsync(dataId string, data ...string) {
    rc.registerWorker.SubmitPublishTask(dataId, data, model.EventTypePb_UNREGISTER.String())
}

func (rc *ConfregClient) SubscribeAsync(dataId string) {
    rc.registerWorker.SubmitSubscribeTask(dataId, model.EventTypePb_REGISTER.String())
}

func (rc *ConfregClient) UnSubscribeAsync(dataId string) {
    rc.registerWorker.SubmitSubscribeTask(dataId, model.EventTypePb_UNREGISTER.String())
}

func (rc *ConfregClient) PublishSync(dataId string, data ...string) error {
    return rc.registerWorker.PublishSync(dataId, data)
}

func (rc *ConfregClient) UnPublishSync(dataId string, data ...string) error {
    return rc.registerWorker.UnPublishSync(dataId, data)
}

func (rc *ConfregClient) SubscribeSync(dataId string) error {
    return rc.registerWorker.SubscribeSync(dataId)
}

func (rc *ConfregClient) UnSubscribeSync(dataId string) error {
    return rc.registerWorker.UnSubscribeSync(dataId)
}