package registry

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "github.com/julienschmidt/httprouter"
    "time"
)

type MsgChannel struct {
    ServiceInfo
    MsgChannelCB   MsgChanCallback
    registryClient RegistryClient
}

type MsgChanCallback func([]string)

type ServiceInfo struct {
    ServiceSet []string `json:"service_set"`
}

var subscribeRecorder = make(map[string]chan bool)
var rpcServerChangeListener servermanager.RPCServerChangeListener

type StartupRPCServerChangeListener struct {
}

func (l *StartupRPCServerChangeListener) OnRPCServerChanged(dataId string, zoneServers map[string][]string) {
    if r, ok := subscribeRecorder[dataId]; ok {
        r <- true
    }
}

func NewMsgChannel(services []string, msgChannelCB MsgChanCallback, registryClient RegistryClient) *MsgChannel {
    rpcServerChangeListener = &StartupRPCServerChangeListener{}
    registryClient.GetRPCServerManager().RegisterRPCServerChangeListener(rpcServerChangeListener)
    return &MsgChannel{
        ServiceInfo:    ServiceInfo{ServiceSet: services},
        MsgChannelCB:   msgChannelCB,
        registryClient: registryClient,
    }
}

func (m *MsgChannel) PublishService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    raw, _ := ioutil.ReadAll(r.Body)

    var request PublishServiceRequest
    if err := json.Unmarshal(raw, &request); err != nil {
        doResponse(false, "Unmarshal body stream to publish service request failed.", w)
        return
    }
    dataId := ps.ByName("serviceName")
    //todo Assemble publish data
    err := m.registryClient.PublishSync(dataId, "127.0.0.1:6666")
    if err == nil {
        doResponse(true, "success", w)
    } else {
        doResponse(false, err, w)
    }
}

func (m *MsgChannel) UnPublishService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    body, _ := ioutil.ReadAll(r.Body)

    var request UnPublishServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to unpublish service request failed.", w)
        return
    }
    err := m.registryClient.UnPublishSync(request.ServiceName)
    if err == nil {
        doResponse(true, "success", w)
    } else {
        doResponse(false, err, w)
    }
}

// 订阅服务

func (m *MsgChannel) SubscribeService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    body, _ := ioutil.ReadAll(r.Body)

    var request SubscribeServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to subscribe service request failed.", w)
        return
    }
    //1. Subscribe from confreg
    dataId := request.ServiceName
    err := m.registryClient.SubscribeSync(dataId)
    if err != nil {
        doResponse(false, err, w)
        return
    }
    //2. Get service info from confreg in block.
    subscribeRecorder[dataId] = make(chan bool)
    timeout := 3 * time.Second
    t := time.NewTimer(timeout)
    for ; ; {
        select {
        case <-t.C:
            {
                doResponse(false, fmt.Sprintf("Wait confreg server push data timeout. data id = %s, timeout = %v", dataId, timeout), w)
                return
            }
        case <-subscribeRecorder[dataId]:
            {
                result := &SubscribeServiceResult{
                    ErrorMessage: "",
                    Success:      true,
                    ServiceName:  dataId,
                }
                servers, ok := m.registryClient.GetRPCServerManager().GetRPCServerList(dataId)
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

// 取消订阅服务

func (m *MsgChannel) UnSubscribeService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    body, _ := ioutil.ReadAll(r.Body)

    var request UnSubscribeServiceRequest
    if err := json.Unmarshal(body, &request); err != nil {
        doResponse(false, "Unmarshal body stream to subscribe service request failed.", w)
        return
    }
    err := m.registryClient.UnSubscribeSync(request.ServiceName)
    if err == nil {
        doResponse(true, "success", w)
    } else {
        doResponse(false, err, w)
    }

}

func checkMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
    if r.Method != expectedMethod {
        doResponse(false, "Unsupported method. The supported method is "+expectedMethod, w)
        return false
    }
    return true
}

func doResponse(success bool, errMsg interface{}, w http.ResponseWriter) {
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

func (m *MsgChannel) StartChannel() {
    router := httprouter.New()
    router.POST("/services/publish", m.PublishService)
    router.DELETE("/services/unpublish", m.UnPublishService)
    router.POST("/services/subscribe", m.SubscribeService)
    router.DELETE("/services/unsubscribe", m.UnSubscribeService)

    port := "8888"
    httpServerEndpoint := "localhost:" + port
    if err := http.ListenAndServe(httpServerEndpoint, router); err != nil {
        log.DefaultLogger.Fatal("Http server startup failed. port = ", port)
    } else {
        log.DefaultLogger.Infof("Http server startup at " + port)
    }
}
