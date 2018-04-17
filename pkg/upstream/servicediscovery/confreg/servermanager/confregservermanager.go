package servermanager

import (
    "strconv"
    "bytes"
    "io/ioutil"
    "net/http"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/javaio"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
)

const (
    REGISTRY_ENDPOINT_PORT = 9603
    ZONE_KEY               = "com.alipay.ldc.zone"
    REGISTRY_VERSION_KEY   = "version"
    REGISTRY_VERSION_VALUE = "4.3.0"
)

type RegistryServerManager struct {
    endpoint        string
    registryServers []string
    changeListeners []RegistryServerChangeListener
}

func NewRegistryServerManager(sysConfig *config.SystemConfig) *RegistryServerManager {
    endpoint := "http://" + sysConfig.RegistryEndpoint + ":" + strconv.Itoa(REGISTRY_ENDPOINT_PORT) + "?" +
        ZONE_KEY + "=" + sysConfig.Zone + "&" +
        REGISTRY_VERSION_KEY + "=" + REGISTRY_VERSION_VALUE

    csm := &RegistryServerManager{
        endpoint:        endpoint,
        changeListeners: make([]RegistryServerChangeListener, 0, 10),
    }

    servers, err := csm.fetchRegistryServer()
    if err != nil {
        log.DefaultLogger.Errorf("Fetch registry server failed. %s, %v", endpoint, err)
    }
    csm.registryServers = servers

    //Refresh endpoint list at fix rate
    //go func() {
    //    for ; ; {
    //        time.Sleep(1 * time.Minute)
    //        newSvrs, _ := csm.fetchRegistryServer()
    //
    //        if isChanged(csm.registryServers, newSvrs) {
    //            csm.triggerChangeEvent(newSvrs)
    //            csm.registryServers = newSvrs
    //        }
    //
    //    }
    //
    //}()

    return csm
}

func (csm *RegistryServerManager) fetchRegistryServer() (result []string, err error) {
    log.DefaultLogger.Debugf("Try fetch registry server list. Confreg endpoint = %v", csm.endpoint)

    fetchConfregServerCommand := []byte{
        0xac, 0xed, 0x00, 0x05, 0x73, 0x72, 0x00, 0x2c, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x6c, 0x69, 0x70, 0x61, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x2e, 0x4e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x03, 0xd0, 0x15, 0x42, 0xd4, 0xca, 0xfc, 0x87, 0x02, 0x00, 0x04, 0x5a, 0x00, 0x0c, 0x69, 0x73, 0x4e, 0x65, 0x77, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x4c, 0x00, 0x02, 0x69, 0x64, 0x74, 0x00, 0x12, 0x4c, 0x6a, 0x61, 0x76, 0x61, 0x2f, 0x6c, 0x61, 0x6e, 0x67, 0x2f, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x3b, 0x4c, 0x00, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x71, 0x00, 0x7e, 0x00, 0x01, 0x5b, 0x00, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x74, 0x00, 0x13, 0x5b, 0x4c, 0x6a, 0x61, 0x76, 0x61, 0x2f, 0x6c, 0x61, 0x6e, 0x67, 0x2f, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x3b, 0x78, 0x70, 0x00, 0x70, 0x74, 0x00, 0x0f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x6c, 0x69, 0x73, 0x74, 0x70}
    commandBytes := bytes.NewBuffer(fetchConfregServerCommand)

    resp, err := http.Post(csm.endpoint, "application/x-java-serialized-object", commandBytes)

    defer resp.Body.Close()

    if err == nil {
        bodyBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            log.DefaultLogger.Errorf("Fetch registry server failed. Confreg endpoint = %v", csm.endpoint)
            panic(err)
        }

        inputObjectStream := javaio.InputObjectStream{}
        inputObjectStream.SetBytes(bodyBytes)
        var result = inputObjectStream.ReadContent()

        if servers, ok := result.(*javaio.List); ok {
            log.DefaultLogger.Infof("Fetch registry server success. Confreg server = %v", servers)

            newSvrs := servers.GetValue()

            return newSvrs, nil
        } else {
            log.DefaultLogger.Errorf("Decode registry server list from java object bytes failed. Confreg endpoint = %v", csm.endpoint)
            panic(err)
        }
    }
    panic(err)
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

func (csm *RegistryServerManager) GetRegistryServerList() ([]string, error) {
    if len(csm.registryServers) > 0 {
        return csm.registryServers, nil
    }
    panic("Registry server list is empty.")
}

func (csm *RegistryServerManager) GetRegistryServerByRandom() (string, error) {
    return "127.0.0.1:8089", nil
    //if len(csm.registryServers) > 0 {
    //    index := rand.Intn(len(csm.registryServers))
    //    return csm.registryServers[index], nil
    //}
    //panic("Registry server list is empty.")
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
