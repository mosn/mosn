package network

import (
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	SO_MARK              = 0x24
	SOL_IP               = 0x0
	IP_TRANSPARENT       = 0x13
)

var sockMarkStore sync.Map

func GetOrCreateAddrMark(address string, mark uint32) {
	if v, ok := sockMarkStore.Load(address); ok {
		if mark == 0 {
			sockMarkStore.Delete(address)
		} else if (v.(uint32) != mark) {
			sockMarkStore.Store(address, int(mark))
		}
	} else {
		if mark > 0 {
			sockMarkStore.Store(address, int(mark))
		}
	}
}

func SockMarkLookup(address string) (int, bool) {
	if v, ok := sockMarkStore.Load(address); ok {
		return v.(int), true
	}
	return 0, false
}

func SockMarkControl(network, address string, c syscall.RawConn) error {
	var err error
	if cerr := c.Control(func(fd uintptr) {
		if mark, ok := SockMarkLookup(address); ok {
			err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, SO_MARK, mark)
			if err != nil {
				return
			}
	}}); cerr != nil {
		return cerr
	}

	if err != nil {
		return err
	}
	return nil
}
