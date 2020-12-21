package network

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func SockMarkControl(network, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, 68)
		if err != nil {
			return
		}
	})
	return err
}
