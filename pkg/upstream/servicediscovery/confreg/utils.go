package registry

import (
    "github.com/nu7hatch/gouuid"
    "time"
    "net"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func RandomUuid() string {
    u, _ := uuid.NewV4()
    return u.String()
}

func CalRetreatTime(t int64, maxTime int64) time.Duration {
    return time.Duration(0)
    //if t == 0 {
    //    return time.Duration(0)
    //}
    //if t > maxTime {
    //    //5min
    //    return 5 * time.Minute
    //}
    //r := 1 << uint(t)
    //return time.Duration(r * 1000 * 1000 * 1000)
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