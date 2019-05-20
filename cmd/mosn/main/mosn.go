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

package main

import (
	_ "flag"
	"os"
	"time"

	_ "sofastack.io/sofa-mosn/pkg/buffer"
	_ "sofastack.io/sofa-mosn/pkg/filter/network/proxy"
	_ "sofastack.io/sofa-mosn/pkg/filter/network/tcpproxy"
	_ "sofastack.io/sofa-mosn/pkg/filter/stream/faultinject"
	_ "sofastack.io/sofa-mosn/pkg/filter/stream/healthcheck/sofarpc"
	_ "sofastack.io/sofa-mosn/pkg/filter/stream/mixer"
	_ "sofastack.io/sofa-mosn/pkg/metrics/sink"
	_ "sofastack.io/sofa-mosn/pkg/metrics/sink/prometheus"
	_ "sofastack.io/sofa-mosn/pkg/network"
	_ "sofastack.io/sofa-mosn/pkg/protocol"
	_ "sofastack.io/sofa-mosn/pkg/protocol/http/conv"
	_ "sofastack.io/sofa-mosn/pkg/protocol/http2/conv"
	_ "sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	_ "sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc/conv"
	_ "sofastack.io/sofa-mosn/pkg/protocol/rpc/xprotocol/tars"
	_ "sofastack.io/sofa-mosn/pkg/router"
	_ "sofastack.io/sofa-mosn/pkg/stream/http"
	_ "sofastack.io/sofa-mosn/pkg/stream/http2"
	_ "sofastack.io/sofa-mosn/pkg/stream/sofarpc"
	_ "sofastack.io/sofa-mosn/pkg/stream/xprotocol"
	_ "sofastack.io/sofa-mosn/pkg/upstream/healthcheck"
	_ "sofastack.io/sofa-mosn/pkg/xds"
	"github.com/urfave/cli"
)

var Version = "0.4.0"

func main() {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = Version
	app.Compiled = time.Now()
	app.Copyright = "(c) 2018 Ant Financial"
	app.Usage = "MOSN is modular observable smart netstub."

	//commands
	app.Commands = []cli.Command{
		cmdStart,
		cmdStop,
		cmdReload,
	}

	//action
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)

		c.App.Setup()
		return nil
	}

	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(os.Args)
}
