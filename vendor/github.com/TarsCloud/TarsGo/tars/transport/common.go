package transport

import "net"

func isNoDataError(err error) bool {
	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() && netErr.Temporary() {
		return true
	}
	return false
}
