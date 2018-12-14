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
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/gogo/protobuf/jsonpb"
)

const (
	PseudoHeaderMethod    = ":method"
	PseudoHeaderPath      = ":path" // indicate method name in rpc protocol
	PseudoHeaderScheme    = ":scheme"
	PseudoHeaderAuthority = ":authority"
)

// parse address like "192.0.2.1:25", "[2001:db8::1]:80" into ip & port
func parseAddr(addr string) (ipAddr net.IP, port int, err error) {
	s := strings.Split(addr, ":")
	if len(s) != 2 {
		return nil, 0, fmt.Errorf("[parseAddr] %s is not a valid address, address must in ip:port format", addr)
	}

	if ipAddr = net.ParseIP(s[0]); ipAddr == nil {
		return nil, 0, fmt.Errorf("[parseAddr] %s is not a valid ip address", s[0])
	}

	if port, err = strconv.Atoi(s[1]); err != nil {
		return nil, 0, fmt.Errorf("[parseAddr] parse port failed, err: %v", err)
	} else {
		if port <= 0 || port > 65535 {
			return nil, 0, fmt.Errorf("[parseAddr] %d not like a valid port", port)
		}
	}

	return ipAddr, port, nil
}

// fetch target value from header, return "" if not found
func headerMapper(target string, headers types.HeaderMap) (string, bool) {
	// TODO: make sure pseudo-header parsing is correct
	switch strings.ToLower(target) {
	case PseudoHeaderMethod:
		return headers.Get("X-Mosn-Method")
	case PseudoHeaderPath:
		return headers.Get("X-Mosn-Path")
	case PseudoHeaderScheme:
		// TODO: parse `:scheme` here
		return "", false
	case PseudoHeaderAuthority:
		return headers.Get("Authority")
	default:
		return headers.Get(target)
	}
}

// parse rbac filter config to v2.RBAC struct
func ParseRbacFilterConfig(cfg map[string]interface{}) (*v2.RBAC, error) {
	filterConfig := new(v2.RBAC)

	jsonConf, err := json.Marshal(cfg)
	if err != nil {
		log.StartLogger.Errorf("parsing rabc filter configuration failed, err: %v, cfg: %v", err, cfg)
		return nil, err
	}

	// parse rules
	var un jsonpb.Unmarshaler
	if err = un.Unmarshal(strings.NewReader(string(jsonConf)), &filterConfig.RBAC); err != nil {
		log.StartLogger.Errorf("parsing rabc filter configuration failed, err: %v, cfg: %v", err, string(jsonConf))
		return nil, err
	}

	return filterConfig, nil
}
