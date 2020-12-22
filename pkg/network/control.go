package network

import (
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	SO_MARK              = 0x24
	SOL_IP               = 0x0
	IP_TRANSPARENT       = 0x13
)

func SockMarkControl(network, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, SO_MARK, 68)
		if err != nil {
			return
		}
	})
	return err
}
