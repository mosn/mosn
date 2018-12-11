package healthcheck

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTCPDial(t *testing.T) {
	s := httptest.NewServer(nil)
	addr := strings.Split(s.URL, "http://")[1]
	host := &mockHost{
		addr: addr,
	}
	dialfactory := &TCPDialSessionFactory{}
	session := dialfactory.NewSession(nil, host)
	if !session.CheckHealth() {
		t.Error("tcp dial check health failed")
	}
	s.Close()
	if session.CheckHealth() {
		t.Error("tcp dial a closed server, but returns ok")
	}
}
