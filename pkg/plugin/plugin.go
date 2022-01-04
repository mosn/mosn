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

package plugin

import (
	"os"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/plugin/proto"
)

var pluginLogBase string

func InitPlugin(log string) {
	if log != "" && log[len(log)-1] != os.PathSeparator {
		log += string(os.PathSeparator)
	}
	pluginLogBase = log
}

func GetLogPath() string {
	if pluginLogBase != "" {
		return pluginLogBase
	}
	return os.Getenv("MOSN_LOGBASE")
}

// Service is a service that Implemented by plugin main process
type Service interface {
	Call(request *proto.Request) (*proto.Response, error)
}

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "MOSN_PLUGIN",
	MagicCookieValue: "ON",
}

var PluginMap = map[string]plugin.Plugin{
	"MOSN_SERVICE": &Plugin{},
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type Plugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Service
}

func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServer(s, &server{Impl: p.Impl})
	return nil
}

func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &client{PluginClient: proto.NewPluginClient(c)}, nil
}
