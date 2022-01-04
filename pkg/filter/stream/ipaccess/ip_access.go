/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ipaccess

import (
	"errors"
	"fmt"
	"net"
)

// IPAccess  check if ip allows access
type IPAccess interface {
	IsAllow(ip string) (bool, error)
	IsDenyAction() bool
}

type IpList struct {
	ips    map[string]*net.IP
	ipNets []*net.IPNet
}

// Exist checks if provided address is in the trusted IPs.
func (i *IpList) Exist(addr string) (bool, error) {
	// fixme ipv4 and ipv6 need to set overlap ip separately
	if addr == "" {
		return false, errors.New("empty IP address")
	}
	if _, ok := i.ips[addr]; ok {
		return true, nil
	}

	ip := net.ParseIP(addr)
	if ip == nil {
		return false, fmt.Errorf("can't parse IP from address %s", addr)
	}
	for _, ipNet := range i.ipNets {
		if ipNet.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}

func NewIpList(addrs []string) (*IpList, error) {
	if len(addrs) == 0 {
		return nil, errors.New("not found trusted IPs")
	}

	ipList := &IpList{
		ips: make(map[string]*net.IP),
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			ipList.ips[addr] = &ip
			continue
		}
		_, ipNet, err := net.ParseCIDR(addr)
		if err != nil {
			return nil, fmt.Errorf("parsing CIDR failed with addr %s: %w", ipNet, err)
		}
		ipList.ipNets = append(ipList.ipNets, ipNet)
	}

	return ipList, nil
}
