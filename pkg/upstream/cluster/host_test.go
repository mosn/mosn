package cluster

import (
	"context"
	"net"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	 "github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestHostDisableTLS(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	defer ln.Close()
	addr := ln.Addr().String()
	// cluster config
	tlsConfig := &v2.TLSConfig{
		Status: true,
	}
	info := &clusterInfo{
		name:                 "test",
		connBufferLimitBytes: 16 * 1026,
	}
	tlsMng, err := mtls.NewTLSClientContextManager(tlsConfig, info)
	if err != nil {
		t.Error(err)
		return
	}
	info.tlsMng = tlsMng
	hosts := []v2.Host{
		{
			Address: addr,
		},
		{
			Address:    addr,
			TLSDisable: true,
		},
	}
	for i, host := range hosts {
		h := NewHost(host, info)
		connData := h.CreateConnection(context.Background())
		conn := connData.Connection
		if err := conn.Connect(false); err != nil {
			t.Errorf("#%d %v", i, err)
			continue
		}
		if _, ok := conn.RawConn().(*mtls.TLSConn); ok == host.TLSDisable {
			t.Errorf("#%d  tlsdisable: %v, conn is tls: %v", i, host.TLSDisable, ok)
		}
		conn.Close(types.NoFlush, types.LocalClose)
	}
}
