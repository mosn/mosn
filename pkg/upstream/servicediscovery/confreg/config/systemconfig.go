package config

import (
    "github.com/magiconair/properties"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "fmt"
    "strings"
    "os"
)

var ServerConfFilePath string

func init() {
    ServerConfFilePath = os.Getenv("server_conf_path")
    if ServerConfFilePath == "" {
        ServerConfFilePath = "/home/admin/server.conf"
    }
}
type SystemConfig struct {
    AntShareCloud    bool
    InstanceId       string
    DataCenter       string
    AppName          string
    Zone             string
    RegistryEndpoint string
    AccessKey        string
    SecretKey        string
}

var SysConfig *SystemConfig

func InitSystemConfig(antShareCloud bool, dc string, appName string, zone string) *SystemConfig {
    if SysConfig != nil {
        return SysConfig
    }

    SysConfig = &SystemConfig{
        AntShareCloud: antShareCloud,
        DataCenter:    dc,
        AppName:       appName,
        Zone:          zone,
        InstanceId:    "",
    }

    confregUrl, z := readPropertyFromServerConfFile(antShareCloud)
    if SysConfig.Zone == "" {
        SysConfig.Zone = z
    }
    if !strings.HasPrefix(confregUrl, "http://") {
        confregUrl = "http://" + confregUrl
    }
    SysConfig.RegistryEndpoint = confregUrl
    return SysConfig
}

func readPropertyFromServerConfFile(antShareCloud bool) (confregUrl string, zone string) {
    if !antShareCloud {
        serverConf := properties.MustLoadFile(ServerConfFilePath, properties.UTF8)
        cu, ok := serverConf.Get("confregurl")
        if !ok {
            errMsg := fmt.Sprintf("Load confregurl from %s failed.", ServerConfFilePath)
            log.DefaultLogger.Errorf(errMsg)
            panic(errMsg)
        }
        z, _ := serverConf.Get("zone")
        return cu, z
    }
    return "", ""
}
