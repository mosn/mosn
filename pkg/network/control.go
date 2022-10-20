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

package network

import (
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	SO_MARK        = 0x24
	SOL_IP         = 0x0
	IP_TRANSPARENT = 0x13
)

var sockMarkStore sync.Map

func GetOrCreateAddrMark(address string, mark uint32) {
	m := int(mark)
	if v, ok := sockMarkStore.Load(address); ok {
		if mark == 0 {
			sockMarkStore.Delete(address)
		} else if v.(int) != m {
			sockMarkStore.Store(address, m)
		}
	} else {
		if mark > 0 {
			sockMarkStore.Store(address, m)
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
		}
	}); cerr != nil {
		return cerr
	}

	if err != nil {
		return err
	}
	return nil
}
