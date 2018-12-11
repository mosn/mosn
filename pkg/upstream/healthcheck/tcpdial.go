package healthcheck

import (
	"net"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type TCPDialSessionFactory struct{}

func (f *TCPDialSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	return &TCPDialSession{
		addr: host.AddressString(),
	}
}

type TCPDialSession struct {
	addr string
}

func (s *TCPDialSession) CheckHealth() bool {
	conn, err := net.Dial("tcp", s.addr)
	if err != nil {
		log.DefaultLogger.Errorf("dial tcp for host %s error: %v", s.addr, err)
		return false
	}
	conn.Close()
	return true
}

func (s *TCPDialSession) OnTimeout() {}
