package stream

import (
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/types"

	"net"
	"testing"
)

func TestStreamOnEvent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestStreamOnEvent error: %v", r)
		}
	}()

	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	con := network.NewClientConnection(nil, 0, nil, remoteAddr, nil)
	if con == nil {
		t.Errorf("NewClientConnection error")
	}

	client := stream.NewStreamClient(nil, protocol.HTTP1, con, nil)
	if client == nil {
		t.Errorf("NewStreamClient error")
	}

	client.OnEvent(types.RemoteClose)
}
