// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxnet encapsulates some network functions
package gxnet

import (
	"net"
	"strconv"
	"strings"
)

// HostAddress composes a ip:port style address. Its opposite function is net.SplitHostPort.
func HostAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func WSHostAddress(host string, port int, path string) string {
	return "ws://" + net.JoinHostPort(host, strconv.Itoa(port)) + path
}

func WSSHostAddress(host string, port int, path string) string {
	return "wss://" + net.JoinHostPort(host, strconv.Itoa(port)) + path
}

func HostAddress2(host string, port string) string {
	return net.JoinHostPort(host, port)
}

func WSHostAddress2(host string, port string, path string) string {
	return "ws://" + net.JoinHostPort(host, port) + path
}

func WSSHostAddress2(host string, port string, path string) string {
	return "wss://" + net.JoinHostPort(host, port) + path
}

func HostPort(addr string) (string, string, error) {
	return net.SplitHostPort(addr)
}

// refers from https://github.com/facebookgo/grace/blob/master/gracenet/net.go#L180:6
func IsSameAddr(a1, a2 net.Addr) bool {
	if a1.Network() != a2.Network() {
		return false
	}
	a1s := a1.String()
	a2s := a2.String()
	if a1s == a2s {
		return true
	}

	// This allows for ipv6 vs ipv4 local addresses to compare as equal. This
	// scenario is common when listening on localhost.
	const ipv6prefix = "[::]"
	a1s = strings.TrimPrefix(a1s, ipv6prefix)
	a2s = strings.TrimPrefix(a2s, ipv6prefix)
	const ipv4prefix = "0.0.0.0"
	a1s = strings.TrimPrefix(a1s, ipv4prefix)
	a2s = strings.TrimPrefix(a2s, ipv4prefix)
	return a1s == a2s
}
