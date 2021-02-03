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

package originaldst

import (
	"errors"
	"fmt"
	"net"
	"syscall"

	"mosn.io/mosn/pkg/log"
)

// OriginDST, option for syscall.GetsockoptIPv6Mreq
const (
	SO_ORIGINAL_DST      = 80
	IP6T_SO_ORIGINAL_DST = 80
)

func getOriginalAddr(conn net.Conn) ([]byte, int, error) {
	tc := conn.(*net.TCPConn)

	f, err := tc.File()
	if err != nil {
		log.DefaultLogger.Errorf("[originaldst] get conn file error, err: %v", err)
		return nil, 0, errors.New("conn has error")
	}
	defer f.Close()

	fd := int(f.Fd())
	addr, err := syscall.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST)

	if err := syscall.SetNonblock(fd, true); err != nil {
		return nil, 0, fmt.Errorf("setnonblock %v", err)
	}

	p0 := int(addr.Multiaddr[2])
	p1 := int(addr.Multiaddr[3])

	port := p0*256 + p1

	ip := addr.Multiaddr[4:8]

	return ip, port, nil
}
