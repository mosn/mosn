///*
// * Licensed to the Apache Software Foundation (ASF) under one or more
// * contributor license agreements.  See the NOTICE file distributed with
// * this work for additional information regarding copyright ownership.
// * The ASF licenses this file to You under the Apache License, Version 2.0
// * (the "License"); you may not use this file except in compliance with
// * the License.  You may obtain a copy of the License at
// *
// *     http://www.apache.org/licenses/LICENSE-2.0
// *
// * Unless required by applicable law or agreed to in writing, software
// * distributed under the License is distributed on an "AS IS" BASIS,
// * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// * See the License for the specific language governing permissions and
// * limitations under the License.
// */
//
package common

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/types"
)

const (
	HeaderMapMethod = ":method"
	HeaderMapPath   = ":path"
)

// parse address like "192.0.2.1:25", "[2001:db8::1]:80" into ip & port
func parseAddr(addr string) (ipAddr net.IP, port int, err error) {
	s := strings.Split(addr, ":")
	if len(s) != 2 {
		return nil, 0, fmt.Errorf("%s is not a valid address, address must in ip:port format", addr)
	}

	if ipAddr = net.ParseIP(s[0]); ipAddr == nil {
		return nil, 0, fmt.Errorf("%s is not a valid ip address", s[0])
	}

	if port, err = strconv.Atoi(s[1]); err != nil {
		return nil, 0, fmt.Errorf("parse port failed, err: %v", err)
	} else {
		if port <= 0 || port > 65535 {
			return nil, 0, fmt.Errorf("%d not like a valid port", port)
		}
	}

	return ipAddr, port, nil
}

// fetch target value from header, return nil if not found
func headerMapper(target string, headers types.HeaderMap) interface{} {
	switch strings.ToLower(target) {
	case HeaderMapMethod:
		if method, ok := headers.Get("X-Mosn-Method"); ok {
			return method
		} else {
			return nil
		}
	case HeaderMapPath:
		if path, ok := headers.Get("X-Mosn-Path"); ok {
			return path
		} else {
			return nil
		}
	default:
		return nil
	}
}
