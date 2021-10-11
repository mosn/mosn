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

package grpc

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type grpcFilter struct {
	ln   *Listener
	conn *Connection
}

var _ api.ReadFilter = (*grpcFilter)(nil)

// TODO: maybe we needs the context in the future
func NewGrpcFilter(_ context.Context, ln *Listener) *grpcFilter {
	return &grpcFilter{
		ln: ln,
	}
}

func (f *grpcFilter) OnData(buf buffer.IoBuffer) api.FilterStatus {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter received a data buffer: %d", buf.Len())
	}
	f.dispatch(buf)
	// grpc filer should be the final network filter.
	return api.Stop
}

func (f *grpcFilter) OnNewConnection() api.FilterStatus {
	// send a grpc connection to Listener to awake Listener's Accept
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter send a new connection to grpc server")
	}
	if err := f.ln.NewConnection(f.conn); err != nil {
		return api.Stop
	}
	return api.Continue
}

func (f *grpcFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter received a new connection: %v", cb.Connection())
	}
	conn := cb.Connection()
	f.conn = NewConn(conn)
}

func (f *grpcFilter) dispatch(buf buffer.IoBuffer) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc get datas: %d", buf.Len())
	}
	// send data to awake connection Read
	f.conn.Send(buf)
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("read dispatch finished")
	}
}
