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

package healthcheck

import (
	"context"
	"net"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type TCPDialSessionFactory struct{}

func (f *TCPDialSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	return &TCPDialSession{
		addr: host.AddressString(),
	}
}

type TCPDialSession struct {
	addr string
}

func (s *TCPDialSession) CheckHealth(ctx context.Context) bool {
	d := net.Dialer{}
	// default dial timeout, maybe already timeout by checker
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[upstream] [health check] [tcpdial session] dial tcp for host %s error: %v", s.addr, err)
		}
		return false
	}
	_ = conn.Close()
	return true
}

func (s *TCPDialSession) OnTimeout() {}
