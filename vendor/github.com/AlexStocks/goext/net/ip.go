// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxnet encapsulates some network functions

package gxnet

import (
	"net"
)

import (
	jerrors "github.com/juju/errors"
)

var (
	privateBlocks []*net.IPNet
)

func init() {
	for _, b := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		if _, block, err := net.ParseCIDR(b); err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func isPrivateIP(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}

// ref: https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func GetLocalIP() (string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", jerrors.Trace(err)
	}

	var ipAddr []byte
	for _, i := range ifs {
		addrs, err := i.Addrs()
		if err != nil {
			return "", jerrors.Trace(err)
		}
		var ip net.IP
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if !ip.IsLoopback() && ip.To4() != nil && isPrivateIP(ip.String()) {
				ipAddr = ip
				break
			}
		}
	}

	if ipAddr == nil {
		return "", jerrors.Errorf("can not get local IP")
	}

	return net.IP(ipAddr).String(), nil
}

// Get preferred outbound ip of this machine
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", jerrors.Trace(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
