package network

import (
	"mosn.io/mosn/pkg/windows"
	"net"
	"os"
)

func transferSendFD(_ *net.UnixConn, _ *os.File) error {
	return windows.ErrUnimplemented
}

func transferRecvFD(_ []byte) (net.Conn, error) {
	return nil, windows.ErrUnimplemented
}
