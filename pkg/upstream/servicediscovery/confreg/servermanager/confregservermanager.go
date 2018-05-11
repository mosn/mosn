package servermanager

import (
    "io/ioutil"
    "net/http"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "time"
    "net"
    "fmt"
    "strings"
    "math/rand"
    "errors"
)


type RegistryServerManager struct {
    sysConfig       *config.SystemConfig
    registryConfig  *config.RegistryConfig
    httpClient      http.Client
    endpoint        string
    registryServers []string
    changeListeners []RegistryServerChangeListener
    rrIndex         int
}

func NewRegistryServerManager(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig) *RegistryServerManager {
    var env string
    if sysConfig.AntShareCloud {
        env = "share"
    } else {
        env = "other"
    }
    endpoint := fmt.Sprintf("%s:%d/api/servers/query?env=%s&zone=%s&dataCenter=%s&appName=%s&instanceId=%s",
        sysConfig.RegistryEndpoint, registryConfig.RegistryEndpointPort, env, sysConfig.Zone, sysConfig.DataCenter,
        sysConfig.AppName, sysConfig.InstanceId)

    csm := &RegistryServerManager{
        sysConfig:       sysConfig,
        registryConfig:  registryConfig,
        endpoint:        endpoint,
        changeListeners: make([]RegistryServerChangeListener, 0, 10),
        rrIndex:         -1,
    }

    httpClient := http.Client{
        Transport: &http.Transport{
            Dial: func(netw, addr string) (net.Conn, error) {
                deadline := time.Now().Add(3 * time.Second)
                c, err := net.DialTimeout(netw, addr, 3*time.Second)
                if err != nil {
                    return nil, err
                }
                c.SetDeadline(deadline)
                return c, nil
            },
        },
    }
    csm.httpClient = httpClient

    for i := 0; i < 3; i++ {
        servers, err := csm.fetchRegistryServer()
        if err != nil || len(servers) == 0 {
            log.DefaultLogger.Errorf("Fetch registry server failed at %d time. %s, %v", i, endpoint, err)
            continue
        }
        //Fetch confreg server success
        go csm.refreshConfregServerAtFixRate()
        csm.registryServers = servers

        return csm
    }
    //Should shutdown application when can not fetch confreg server list.
    panic("Can not fetch confreg server list from endpoint.")
}

func (csm *RegistryServerManager) fetchRegistryServer() (serverList []string, err error) {
    resp, err := http.Get(csm.endpoint)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.DefaultLogger.Errorf("Fetch confreg server failed. endpoint = %s", csm.endpoint)
        return nil, err
    }
    srvStr := string(bodyBytes)
    if srvStr == "" {
        log.DefaultLogger.Warnf("Fetched empty confreg server. endpoint = %s", csm.endpoint)
        return nil, errors.New("empty server list")
    } else {
        log.DefaultLogger.Infof("Fetched confreg server list = %s, endpoint = %s", srvStr, csm.endpoint)
        return strings.Split(srvStr, ";"), nil
    }
}

func isChanged(oldSvrs []string, newSvrs []string) bool {

    if len(newSvrs) != len(oldSvrs) {
        return true
    }
    newSvrMap := make(map[string]bool)
    for _, svr := range newSvrs {
        newSvrMap[svr] = true
    }
    for _, svr := range oldSvrs {
        _, ok := newSvrMap[svr]
        if !ok {
            return true
        }
    }
    return false
}

func (csm *RegistryServerManager) GetRegistryServerList() ([]string, bool) {
    if len(csm.registryServers) > 0 {
        return csm.registryServers, true
    }
    return nil, false
}

func (csm *RegistryServerManager) GetRegistryServerByRandom() (string, bool) {
    //return "127.0.0.1:8089", true
    if len(csm.registryServers) > 0 {
        index := rand.Intn(len(csm.registryServers))
        return csm.registryServers[index], true
    }
    return "", false
}

func (csm *RegistryServerManager) GetRegistryServerByRR() (string, bool) {
    serverCount := len(csm.registryServers)
    if serverCount <= 0 {
        return "", false
    }
    csm.rrIndex = csm.rrIndex + 1
    if csm.rrIndex >= serverCount {
        csm.rrIndex = 0
    }
    return csm.registryServers[csm.rrIndex], true
}

func (csm *RegistryServerManager) RegisterServerChangeListener(listener RegistryServerChangeListener) {
    csm.changeListeners = append(csm.changeListeners, listener)
}

func (csm *RegistryServerManager) triggerChangeEvent(newRegistryServer []string) {
    if len(csm.changeListeners) == 0 {
        return
    }
    for _, l := range csm.changeListeners {
        l.OnRegistryServerChangeEvent(newRegistryServer)
    }
}

func (csm *RegistryServerManager) refreshConfregServerAtFixRate() {
    for ; ; {
        time.Sleep(csm.registryConfig.ScheduleRefreshConfregServerTaskDuration)
        log.DefaultLogger.Infof("Start to refresh confreg server list.")
        if newSvrs, err := csm.fetchRegistryServer(); err == nil {
            if isChanged(csm.registryServers, newSvrs) {
                //Update cached confreg server list
                if newSvrs != nil && len(newSvrs) > 0 {
                    csm.registryServers = newSvrs
                }
                csm.triggerChangeEvent(newSvrs)
            }
        }
    }
}
