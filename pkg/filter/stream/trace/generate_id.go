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

package trace

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	TraceIDGeneratorUpperBound uint32 = 8000
	TraceIDGeneratorLowerBound uint32 = 1000

	MaxPid = 0xffff

	OSPid   int
	traceid uint32
	LocalIp string
)

func init() {
	OSPid = os.Getpid()
	if OSPid > MaxPid {
		OSPid &= MaxPid
	}
	LocalIp, _ = getLocalIp()
}

func GetNextTraceID() uint32 {
	return atomic.AddUint32(&traceid, 1)%TraceIDGeneratorUpperBound + TraceIDGeneratorLowerBound
}

type TraceIDGenerator struct {
	ip    string
	hexip []byte
}

func NewTraceIDGenerator(ip string) (*TraceIDGenerator, error) {
	if len(ip) == 0 || ip == "" {
		ip = LocalIp
	}

	t := &TraceIDGenerator{
		ip: ip,
	}
	if err := t.polyfill(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TraceIDGenerator) polyfill() error {
	ip := net.ParseIP(t.ip)
	if v4 := ip.To4(); v4 != nil {
		t.hexip = append(t.hexip, make([]byte, hex.EncodedLen(len(v4)))...)
		n := hex.Encode(t.hexip, []byte(v4))
		t.hexip = t.hexip[:n]
		return nil
	} else {
		return errors.New("not valid IPv4 address")
	}
}

func (tc *TraceIDGenerator) GetHexIP() []byte {
	return tc.hexip
}

// generate new traceid
func (tc *TraceIDGenerator) AppendBytes(d []byte) []byte {
	// ip(8) + timestamp(13) + countid(4+1) + pid(4)
	d = append(d, tc.hexip...)                                       // 8 byte
	d = append(d, []byte(fmt.Sprintf("%013d", unixMilli()))...)      // 13 byte
	d = append(d, []byte(fmt.Sprintf("%04de", GetNextTraceID()))...) // 5 byte
	d = append(d, []byte(fmt.Sprintf("%04x", OSPid))...)             // 4 byte
	return d
}

func (tc *TraceIDGenerator) Bytes() []byte {
	return tc.AppendBytes(nil)
}

func (tc *TraceIDGenerator) String() string {
	return string(tc.Bytes())
}

// unixMilli returns a Unix time, the number of milliseconds elapsed since January 1, 1970 UTC.
func unixMilli() int64 {
	scale := int64(time.Millisecond / time.Nanosecond)
	return time.Now().UnixNano() / scale
}

func getLocalIp() (string, error) {
	faces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var addr net.IP
	for _, face := range faces {
		if !isValidNetworkInterface(face) {
			continue
		}

		addrs, err := face.Addrs()
		if err != nil {
			return "", err
		}

		if ipv4, ok := getValidIPv4(addrs); ok {
			addr = ipv4
		}
	}

	if addr == nil {
		return "", errors.New("can not get local IP")
	}

	return addr.String(), nil
}

func getValidIPv4(addrs []net.Addr) (net.IP, bool) {
	for _, addr := range addrs {
		var ip net.IP

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil || ip.IsLoopback() {
			continue
		}

		ip = ip.To4()
		if ip == nil {
			// not an valid ipv4 address
			continue
		}

		return ip, true
	}
	return nil, false
}

func isValidNetworkInterface(face net.Interface) bool {
	if face.Flags&net.FlagUp == 0 {
		// interface down
		return false
	}

	if face.Flags&net.FlagLoopback != 0 {
		// loopback interface
		return false
	}

	if strings.Contains(strings.ToLower(face.Name), "docker") {
		return false
	}

	return true
}
