package registry

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "github.com/julienschmidt/httprouter"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
)

const ModuleNotStartedErrMsg = "Registry module not startup. Should call '/configs/application' endpoint at first."

func init() {
    rpcServerChangeListener := &WaitDataPushedListener{}
    servermanager.GetRPCServerManager().RegisterRPCServerChangeListener(rpcServerChangeListener)
}

type WaitDataPushedListener struct {
}

func (l *WaitDataPushedListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    if r, ok := subscribeRecorder[dataId]; ok {
        r <- true
    }
}

type Endpoint struct {
    registryConfig *config.RegistryConfig
}

type MsgChanCallback func([]string)

type ServiceInfo struct {
    ServiceSet []string `json:"service_set"`
}

var subscribeRecorder = make(map[string]chan bool)

func (re *Endpoint) SetSystemConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    raw, _ := ioutil.ReadAll(r.Body)
    
    var request ApplicationInfoRequest
    if err := json.Unmarshal(raw, &request); err != nil {
        doResponse(false, fmt.Sprintf("Unmarshal body stream to application info request failed. error = %v", err), w)
        return
    }
    
    sysConfig := config.ForceInitSystemConfig(request.AntShareCloud, request.DataCenter, request.AppName, request.Zone)
    
    //Should reset registry module at first.
    if err := ResetRegistryModule(sysConfig); err != nil {
        log.DefaultLogger.Errorf("Reset registry module failed. error = %v", err)
    } else {
        log.DefaultLogger.Infof("Reset registry module succeeded.")
    }
    
    //Startup registry module.
    StartupRegistryModule(sysConfig, config.DefaultRegistryConfig)
    
    doResponse(true, "", w)
}

func (re *Endpoint) PublishService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    
    raw, _ := ioutil.ReadAll(r.Body)
    
    var request PublishServiceRequest
    if err := json.Unmarshal(raw, &request); err != nil {
        doResponse(false, "Unmarshal body stream to publish service request failed.", w)
        return
    }
    
    pubData := re.assemblePublishData(request)
    //1. Store publish info.
    AddPubInfo(map[string]string{request.ServiceName: pubData})
    //2. Publish data to registry server.
    err := RegistryClient.PublishSync(request.ServiceName, pubData)
    if err == nil {
        doResponse(true, "", w)
    } else {
        doResponse(false, err.Error(), w)
    }
}

func (re *Endpoint) assemblePublishData(request PublishServiceRequest) string {
    data := fmt.Sprintf("%s:%d?_TIMEOUT=3000&p=%s&_SERIALIZETYPE=%s&app_name=%s&zone=%s&v=%s",
        GetOutboundIP().String(), re.registryConfig.RPCServerPort, request.ProviderMetaInfo.Protocol,
        request.ProviderMetaInfo.SerializeType, request.ProviderMetaInfo.AppName,
        config.SysConfig.Zone, request.ProviderMetaInfo.Version)
    return data
}

func (re *Endpoint) UnPublishService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    
    body, _ := ioutil.ReadAll(r.Body)
    
    var request UnPublishServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to unpublish service request failed.", w)
        return
    }
    
    //1. Delete stored publish info.
    DelPubInfo(request.ServiceName)
    
    //2. Unpublish from registry server.
    err := RegistryClient.UnPublishSync(request.ServiceName)
    if err == nil {
        doResponse(true, "", w)
    } else {
        doResponse(false, err.Error(), w)
    }
}

func (re *Endpoint) SubscribeService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    body, _ := ioutil.ReadAll(r.Body)
    
    var request SubscribeServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to subscribe service request failed.", w)
        return
    }
    //1. Subscribe from confreg
    dataId := request.ServiceName
    subscribeRecorder[dataId] = make(chan bool)

    defer func() {
        delete(subscribeRecorder, dataId)
    }()

    err := RegistryClient.SubscribeSync(dataId)
    if err != nil {
        doResponse(false, err.Error(), w)
        return
    }
    
    //2. Store subscribe info to config.
    AddSubInfo([]string{dataId})
    
    //3. Get service info from confreg in block.
    t := time.NewTimer(re.registryConfig.WaitReceivedDataTimeout)
    for ; ; {
        select {
        case <-t.C:
            {
                doResponse(false, fmt.Sprintf("Wait confreg server push data timeout. data id = %s, timeout = %v",
                    dataId, re.registryConfig.WaitReceivedDataTimeout), w)
                return
            }
        case <-subscribeRecorder[dataId]:
            {
                result := &SubscribeServiceResult{
                    ErrorMessage: "",
                    Success:      true,
                    ServiceName:  dataId,
                }
                servers, ok := RegistryClient.GetRPCServerManager().GetRPCServerList(dataId)
                if !ok {
                    result.Datas = []string{}
                } else {
                    for _, v := range servers {
                        result.Datas = v
                        break
                    }
                }
                res, _ := json.Marshal(result)
                w.Write(res)
                return
            }
            
        }
    }
}

func (re *Endpoint) UnSubscribeService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    log.DefaultLogger.Debugf("unsubscribe service called")
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    body, _ := ioutil.ReadAll(r.Body)
    
    var request UnSubscribeServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to subscribe service request failed.", w)
        return
    }
    
    //Delete stored subscribe info.
    DelSubInfo([]string{request.ServiceName})
    
    err := RegistryClient.UnSubscribeSync(request.ServiceName)
    if err == nil {
        doResponse(true, "", w)
    } else {
        doResponse(false, err.Error(), w)
    }
}

func (re *Endpoint) GetSystemConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    res, _ := json.Marshal(config.SysConfig)
    w.Write(res)
}

func (re *Endpoint) GetRegistryConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    res, _ := json.Marshal(config.DefaultRegistryConfig)
    w.Write(res)
}

func (re *Endpoint) GetServiceInfoSnapshot(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    w.Write(RegistryClient.GetRPCServerManager().GetRPCServiceSnapshot())
}

func (re *Endpoint) GetServiceInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    
    dataId := ps.ByName("serviceName")
    services, ok := RegistryClient.GetRPCServerManager().GetRPCServerList(dataId)
    if !ok {
        w.Write([]byte("The services is empty."))
    } else {
        res, _ := json.Marshal(services)
        w.Write(res)
    }
}

func doResponse(success bool, errMsg string, w http.ResponseWriter) {
    if !success {
        log.DefaultLogger.Errorf("Handle http request failed. error message = %v", errMsg)
    }
    r := &HttpResponse{
        Success:      success,
        ErrorMessage: errMsg,
    }
    
    bytes, _ := json.Marshal(r)
    w.Write(bytes)
}

func (re *Endpoint) StartListener() {
    router := httprouter.New()
    router.POST("/configs/application", re.SetSystemConfig)
    router.GET("/configs/application", re.GetSystemConfig)
    router.GET("/configs/registry", re.GetRegistryConfig)
    router.POST("/services/publish", re.PublishService)
    router.POST("/services/unpublish", re.UnPublishService)
    router.POST("/services/subscribe", re.SubscribeService)
    router.POST("/services/unsubscribe", re.UnSubscribeService)
    router.GET("/services", re.GetServiceInfoSnapshot)
    router.GET("/services/:serviceName", re.GetServiceInfo)
    
    port := "8888"
    httpServerEndpoint := "0.0.0.0:" + port
    log.DefaultLogger.Infof("Mesh registry endpoint started on port(s): %s (http)", port)
    
    if err := http.ListenAndServe(httpServerEndpoint, router); err != nil {
        log.DefaultLogger.Fatal("Http server startup failed. port = ", port)
    }
}