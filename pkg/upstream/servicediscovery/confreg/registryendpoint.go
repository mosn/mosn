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
)

const ModuleNotStartedErrMsg =  "Registry module not startup. Should call '/configs/application' endpoint at first."

type Endpoint struct {
    registryConfig *config.RegistryConfig
    RegistryClient Client
}

type MsgChanCallback func([]string)

type ServiceInfo struct {
    ServiceSet []string `json:"service_set"`
}

var subscribeRecorder = make(map[string]chan bool)

type WaitDataPushedListener struct {
}

func (l *WaitDataPushedListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    if r, ok := subscribeRecorder[dataId]; ok {
        r <- true
    }
}

func NewRegistryEndpoint(registryConfig *config.RegistryConfig, registryClient Client) *Endpoint {
    re := &Endpoint{
        registryConfig: registryConfig,
    }
    if registryClient != nil {
        re.RegistryClient = registryClient
    }
    return re
}

func (re *Endpoint) SetSystemConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    raw, _ := ioutil.ReadAll(r.Body)

    var request ApplicationInfoRequest
    if err := json.Unmarshal(raw, &request); err != nil {
        doResponse(false, fmt.Sprintf("Unmarshal body stream to application info request failed. error = %v", err), w)
        return
    }

    sysConfig := config.InitSystemConfig(request.AntShareCloud, request.DataCenter, request.AppName, request.Zone)

    re.RegistryClient = StartupRegistryModule(sysConfig, config.DefaultRegistryConfig)

    rpcServerChangeListener := &WaitDataPushedListener{}
    re.RegistryClient.GetRPCServerManager().RegisterRPCServerChangeListener(rpcServerChangeListener)

    doResponse(true, "", w)
}

func (re *Endpoint) GetSystemConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    res, _ := json.Marshal(config.SysConfig)
    w.Write(res)
}

func (re *Endpoint) GetRegistryConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    res, _ := json.Marshal(config.DefaultRegistryConfig)
    w.Write(res)
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
    //todo Assemble publish data
    err := re.RegistryClient.PublishSync(request.ServiceName, re.assemblePublishData(request))
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
    err := re.RegistryClient.UnPublishSync(request.ServiceName)
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
    err := re.RegistryClient.SubscribeSync(dataId)
    if err != nil {
        doResponse(false, err.Error(), w)
        return
    }
    //2. Get service info from confreg in block.
    subscribeRecorder[dataId] = make(chan bool)
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
                servers, ok := re.RegistryClient.GetRPCServerManager().GetRPCServerList(dataId)
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
    err := re.RegistryClient.UnSubscribeSync(request.ServiceName)
    if err == nil {
        doResponse(true, "", w)
    } else {
        doResponse(false, err.Error(), w)
    }
}

func (re *Endpoint) GetServiceInfoSnapshot(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    if !ModuleStarted {
        doResponse(false, ModuleNotStartedErrMsg, w)
        return
    }
    w.Write(re.RegistryClient.GetRPCServerManager().GetRPCServiceSnapshot())
}

func (re *Endpoint) GetServiceInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    
    dataId := ps.ByName("serviceName")
    services, ok := re.RegistryClient.GetRPCServerManager().GetRPCServerList(dataId)
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
    router.POST("/services/publish", re.PublishService)
    router.POST("/configs/application", re.SetSystemConfig)
    router.GET("/configs/application", re.GetSystemConfig)
    router.GET("/configs/registry", re.GetRegistryConfig)
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
