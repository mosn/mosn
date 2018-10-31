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

package cluster

import (
	"context"
	"net"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestHostDisableTLS(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	defer ln.Close()
	addr := ln.Addr().String()
	// cluster config
	tlsConfig := &v2.TLSConfig{
		Status: true,
	}
	info := &clusterInfo{
		name:                 "test",
		connBufferLimitBytes: 16 * 1026,
	}
	tlsMng, err := mtls.NewTLSClientContextManager(tlsConfig, info)
	if err != nil {
		t.Error(err)
		return
	}
	info.tlsMng = tlsMng
	hosts := []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: addr,
			},
		},
		{
			HostConfig: v2.HostConfig{
				Address:    addr,
				TLSDisable: true,
			},
		},
	}
	for i, host := range hosts {
		h := NewHost(host, info)
		connData := h.CreateConnection(context.Background())
		conn := connData.Connection
		if err := conn.Connect(false); err != nil {
			t.Errorf("#%d %v", i, err)
			continue
		}
		if _, ok := conn.RawConn().(*mtls.TLSConn); ok == host.TLSDisable {
			t.Errorf("#%d  tlsdisable: %v, conn is tls: %v", i, host.TLSDisable, ok)
		}
		conn.Close(types.NoFlush, types.LocalClose)
	}
}
