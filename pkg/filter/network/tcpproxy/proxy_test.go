package tcpproxy

import (
	"net"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func Test_IpRangeList_Contains(t *testing.T) {
	ipRangeList := &IpRangeList{
		cidrRanges: []v2.CidrRange{
			*v2.Create("127.0.0.1", 24),
		},
	}
	httpAddr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !ipRangeList.Contains(httpAddr) {
		t.Errorf("test  ip range fail")
	}
}

func Test_ParsePortRangeList(t *testing.T) {
	prList := ParsePortRangeList("80,443,8080-8089")
	httpPort := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 80,
	}
	if !prList.Contains(httpPort) {
		t.Errorf("test http port fail")
	}
	randomPort := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8083,
	}
	if !prList.Contains(randomPort) {
		t.Errorf("test  port range fail")
	}
}
