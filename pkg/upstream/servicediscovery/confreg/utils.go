package registry

import (
    "github.com/nu7hatch/gouuid"
    "time"
    "net"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "strings"
)

const DataIdSuffix = "@DEFAULT"

func RandomUuid() string {
    u, _ := uuid.NewV4()
    return u.String()
}

func CalRetreatTime(t int64, maxTime int64) time.Duration {
    if t == 0 {
        return time.Duration(0)
    }
    if t > maxTime {
        //5min
        return 5 * time.Minute
    }
    r := 1 << uint(t)
    return time.Duration(r * 1000 * 1000 * 1000)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.DefaultLogger.Errorf("Get local address IP failed.", err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP
}

func appendDataIdSuffix(dataId string) string {
    if strings.HasSuffix(dataId, DataIdSuffix) {
        return dataId
    }
    return dataId + DataIdSuffix
}

func cutoffDataIdSuffix(dataId string) string {
    if strings.HasSuffix(dataId, DataIdSuffix) {
        return dataId[:len(dataId) - 8]
    }
    return dataId
}

