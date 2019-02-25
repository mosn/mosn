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

package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Server struct {
	ln net.Listener
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())

	switch method {
	case http.MethodGet:
		getAPIs(ctx)
	case http.MethodPost:
		postAPIs(ctx)
	default:
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
	}
}

func getAPIs(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	switch path {
	case "/api/v1/config_dump":
		configDump(ctx)

	case "/api/v1/stats":
		statsDump(ctx)
	default:
		ctx.SetStatusCode(404)
	}
}
func postAPIs(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	switch path {
	case "/api/v1/logging":
		setLogLevel(ctx)
	default:
		ctx.SetStatusCode(404)
	}
}

func (s *Server) Start(config Config) {
	var addr string
	if config != nil {
		// merge MOSNConfig into global context
		store.SetMOSNConfig(config)
		// get admin config
		adminConfig := config.GetAdmin()
		if adminConfig == nil {
			// no admin config, no admin start
			log.DefaultLogger.Warnf("no admin config, no admin api served")
			return
		}
		address := adminConfig.GetAddress()
		if xdsPort, ok := address.GetSocketAddress().GetPortSpecifier().(*core.SocketAddress_PortValue); ok {
			addr = fmt.Sprintf("%s:%d", address.GetSocketAddress().GetAddress(), xdsPort.PortValue)
		}
	}

	srv := &fasthttp.Server{
		Handler: requestHandler,
		Name:    "Mosn Admin Server",
	}

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		log.DefaultLogger.Errorf("Admin server: Listen on %s with error: %s\n", addr, err)
		return
	}
	log.DefaultLogger.Infof("Admin server serve on %s\n", addr)
	s.ln = ln
	go func() {
		store.AddStoppable(s)
		if err := srv.Serve(ln); err != nil {
			log.DefaultLogger.Errorf("Admin server: Served) with error: %s\n", err)
		}

	}()
}

func (s *Server) Close() error {
	ln := s.ln
	if ln != nil {
		s.ln = nil
		log.DefaultLogger.Infof("Admin server stopped\n")
		return ln.Close()
	}
	return nil
}
