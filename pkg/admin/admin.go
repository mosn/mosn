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

func configDump(ctx *fasthttp.RequestCtx) {
	if buf, err := json.Marshal(GetEffectiveConfig()); err == nil {
		ctx.Write(buf)
	} else {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(`{ error: "internal error" }`))
		log.DefaultLogger.Errorf("Admin API: ConfigDump failed, cause by %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	path := ctx.Path()
	switch {
	case string(path) == "/api/v1/config_dump":
		configDump(ctx)
	default:
		ctx.SetStatusCode(404)
	}
}

func Start(config Config) net.Listener {
	addr := ":8888"
	if config != nil {
		// merge MOSNConfig into global context
		Set("original_config", config)
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
	log.DefaultLogger.Infof("Admin server serve on %s", addr)

	if err != nil {
		log.DefaultLogger.Errorf("Admin server: Listen on %s with error: %s", addr, err)
	} else {
		go func() {
			if err := srv.Serve(ln); err != nil {
				log.DefaultLogger.Errorf("Admin server: Served) with error: %s", err)
			}

		}()
	}
	return ln
}
