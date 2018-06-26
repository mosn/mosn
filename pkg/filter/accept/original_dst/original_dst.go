package original_dst

import (
    "net"
    "syscall"
    "errors"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    
)

const (
	SO_ORIGINAL_DST      = 80
	IP6T_SO_ORIGINAL_DST = 80
)


type original_dst struct {

}

func NewOriginalDst() Original_Dst{
    return &original_dst{}
}


func (filter *original_dst)OnAccept(cb types.ListenerFilterCallbacks) types.FilterStatus{
    ip, port, err := getOriginalAddr(cb.Conn())
    if(err != nil){
        log.StartLogger.Println("get original addr failed:", err.Error())
        return types.Continue
    }

    ips := string(ip)

    cb.SetOrigingalAddr(ips, port)


    return types.Continue
}



func getOriginalAddr(conn net.Conn)([]byte, int, error){
        tc := conn.(*net.TCPConn)

        f, err := tc.File()
        if err != nil{
            log.StartLogger.Println("get conn file error, err:", err)
            return nil, 0, errors.New("conn has error")
        }


        fd := int(f.Fd())
        addr, err := syscall.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST)
       

        p0 := addr.Multiaddr[2]
        p1 := addr.Multiaddr[3]

        var port uint16 = uint16(p0 * 16 + p1)

        ip := addr.Multiaddr[4:8]
 

        return ip,  int(port), nil
}
