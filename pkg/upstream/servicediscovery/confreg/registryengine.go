package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "sync"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
    "gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

var confregServerManager *servermanager.RegistryServerManager
var RegistryClient Client

var lock = new(sync.Mutex)

var ModuleStarted = false

//Setup registry module.
func init() {
    log.InitDefaultLogger("", log.INFO)
    //RPC server manager is global unique.
    cf := &confregAdaptor{
        ca: &cluster.ClusterAdap,
    }
    servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(cf)
}

func RecoverRegistryModule() {
    defer func() {
        //Start registry endpoint.
        go func() {
            re := &Endpoint{
                registryConfig: config.DefaultRegistryConfig,
            }
            re.StartListener()
        }()
    }()

    appInfo := GetAppInfoFromConfig()
    //Mesh never started before.
    if appInfo.AppName == "" {
        return
    }

    log.DefaultLogger.Infof("Start recover registry module. app name = %v, dc = %v, zone = %v",
        appInfo.AppName, appInfo.DataCenter, appInfo.Zone)

    sysConfig := config.InitSystemConfig(appInfo.AntShareCloud, appInfo.DataCenter, appInfo.AppName, appInfo.Zone)

    RegistryClient = StartupRegistryModule(sysConfig, config.DefaultRegistryConfig)

    //resubscribe
    subs := GetSubListFromConfig()
    if len(subs) > 0 {
        for _, sub := range subs {
            log.DefaultLogger.Infof("Resubscribe service. service name = %v", sub)
            RegistryClient.SubscribeAsync(sub)
        }
    }
    //republish
    pubs := GetPubListFromConfig()
    if len(pubs) > 0 {
        for serviceName, data := range pubs {
            log.DefaultLogger.Infof("Republish service. service name = %v, data = %v", serviceName, data)
            RegistryClient.PublishAsync(serviceName, data)
        }
    }
}

func StartupRegistryModule(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig) Client {
    lock.Lock()

    defer func() {
        lock.Unlock()
    }()

    if ModuleStarted {
        return RegistryClient
    }

    confregServerManager = servermanager.NewRegistryServerManager(sysConfig, registryConfig)
    RegistryClient = NewConfregClient(sysConfig, registryConfig, confregServerManager)

    ModuleStarted = true
    return RegistryClient
}

func ResetRegistryModule(sysConfig *config.SystemConfig) error {
    lock.Lock()

    defer func() {
        lock.Unlock()
    }()

    appInfo := v2.ApplicationInfo{
        AntShareCloud: sysConfig.AntShareCloud,
        DataCenter:    sysConfig.DataCenter,
        AppName:       sysConfig.AppName,
        Zone:          sysConfig.Zone,
    }

    ResetRegistryInfo(appInfo)

    if !ModuleStarted {
        return nil
    }

    RegistryClient.Reset()
    servermanager.GetRPCServerManager().Reset()

    return nil
}
