// +build windows

package grace

type handlerFunc func()

// GraceHandler is not supported in windows
func GraceHandler(stopFunc, userFunc handlerFunc) {
}

// SignalUSR2 is not supported in windows
func SignalUSR2(pid int) {
}
