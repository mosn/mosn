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

package admin

import (
	"fmt"
	"net"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Server struct {
	ln net.Listener
}

func configDump(ctx *fasthttp.RequestCtx) {
	if buf, err := Dump(); err == nil {
		ctx.Write(buf)
	} else {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(`{ error: "internal error" }`))
		log.DefaultLogger.Errorf("Admin API: ConfigDump failed, cause by %s", err)
	}
}

var levelMap = map[string]log.Level{
	"FATAL": log.FATAL,
	"ERROR": log.ERROR,
	"WARN":  log.WARN,
	"INFO":  log.INFO,
	"DEBUG": log.DEBUG,
	"TRACE": log.TRACE,
}

func setLogLevel(ctx *fasthttp.RequestCtx) {
	body := string(ctx.Request.Body())
	if level, ok := levelMap[body]; ok {
		log.DefaultLogger.Level = level
		log.DefaultLogger.Infof("DefaultLogger level has been changed to %s", body)
	} else {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(`{ error: "unknown log level" }`))
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	method := string(ctx.Method())

	switch {
	case path == "/api/v1/config_dump" && method == "GET":
		configDump(ctx)
	case path == "/api/v1/logging" && method == "POST":
		setLogLevel(ctx)
	default:
		ctx.SetStatusCode(404)
	}
}

func (server *Server) Start(config Config) {
	addr := ":8888"
	if config != nil {
		// merge MOSNConfig into global context
		SetMOSNConfig(config)
		// get admin config
		adminConfig := config.GetAdmin()
		if adminConfig != nil {
			address := adminConfig.GetAddress()
			if xdsPort, ok := address.GetSocketAddress().GetPortSpecifier().(*core.SocketAddress_PortValue); ok {
				addr = fmt.Sprintf("%s:%d", address.GetSocketAddress().GetAddress(), xdsPort.PortValue)
			}
		}
	}

	srv := &fasthttp.Server{
		Handler: requestHandler,
		Name:    "Mosn Admin Server",
	}

	ln, err := net.Listen("tcp4", addr)
	log.DefaultLogger.Infof("Admin server serve on %s\n", addr)
	server.ln = ln

	if err != nil {
		log.DefaultLogger.Errorf("Admin server: Listen on %s with error: %s\n", addr, err)
	} else {
		go func() {
			if err := srv.Serve(ln); err != nil {
				log.DefaultLogger.Errorf("Admin server: Served) with error: %s\n", err)
			}

		}()
	}
}

func (server *Server) Close() {
	if server.ln != nil {
		server.ln.Close()
		log.DefaultLogger.Infof("Admin server stopped\n")
	}
	server.ln = nil
}
