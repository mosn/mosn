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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	_ "mosn.io/envoy-go-extension/pkg/filter/stream/echo"
	"mosn.io/envoy-go-extension/pkg/http"
	_ "mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"

	_ "mosn.io/mosn/pkg/admin/debug"
	_ "mosn.io/mosn/pkg/filter/stream/dsl"
	_ "mosn.io/mosn/pkg/filter/stream/dubbo"
	_ "mosn.io/mosn/pkg/filter/stream/faultinject"
	_ "mosn.io/mosn/pkg/filter/stream/faulttolerance"
	_ "mosn.io/mosn/pkg/filter/stream/flowcontrol"
	_ "mosn.io/mosn/pkg/filter/stream/grpcmetric"
	_ "mosn.io/mosn/pkg/filter/stream/gzip"
	_ "mosn.io/mosn/pkg/filter/stream/headertometadata"
	_ "mosn.io/mosn/pkg/filter/stream/ipaccess"
	_ "mosn.io/mosn/pkg/filter/stream/mirror"
	_ "mosn.io/mosn/pkg/filter/stream/payloadlimit"
	_ "mosn.io/mosn/pkg/filter/stream/proxywasm"
	_ "mosn.io/mosn/pkg/filter/stream/seata"
	_ "mosn.io/mosn/pkg/filter/stream/transcoder/httpconv"
	_ "mosn.io/mosn/pkg/metrics/sink"
	_ "mosn.io/mosn/pkg/metrics/sink/prometheus"
	"mosn.io/mosn/pkg/moe"
	_ "mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/protocol/xprotocol"
	_ "mosn.io/mosn/pkg/router"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/streamfilter"
	_ "mosn.io/mosn/pkg/trace/jaeger"
	_ "mosn.io/mosn/pkg/trace/skywalking"
	_ "mosn.io/mosn/pkg/trace/skywalking/http"
	_ "mosn.io/mosn/pkg/trace/sofa/http"
	_ "mosn.io/mosn/pkg/trace/sofa/xprotocol"
	_ "mosn.io/mosn/pkg/upstream/healthcheck"
	_ "mosn.io/mosn/pkg/upstream/servicediscovery/dubbod"
	_ "mosn.io/mosn/pkg/wasm/abi/proxywasm010"
	_ "mosn.io/mosn/pkg/wasm/abi/proxywasm020"
	_ "mosn.io/mosn/pkg/wasm/runtime/wasmer"
	_ "mosn.io/mosn/pkg/wasm/runtime/wazero"
)

// Version mosn version is specified by build tag, in VERSION file
var (
	Version               string = ""
	DefaultMosnConfigPath string = "/home/admin/mosn/config/mosn.json"
)

func init() {
	http.RegisterHttpFilterConfigFactory(moe.ConfigFactory)

	// load mosn config
	if envPath := os.Getenv("MOSN_CONFIG"); envPath != "" {
		DefaultMosnConfigPath = envPath
	}
	streamfilter.LoadAndRegisterStreamFilters(DefaultMosnConfigPath)

	utils.GoWithRecover(func() {
		start()
	}, nil)
}

func start() {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = Version
	app.Compiled = time.Now()
	app.Copyright = "(c) " + strconv.Itoa(time.Now().Year()) + " Ant Group"
	app.Usage = "MOSN is modular observable smart netstub."

	//commands
	app.Commands = []cli.Command{
		cmdStart,
		cmdStop,
		cmdReload,
	}

	//action
	app.Action = func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			return cli.ShowAppHelp(c)
		}
		c.App.Setup()
		return nil
	}

	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(generateArgs())
}

const mosnCmd = "mosn start"

func addCmdArg(cmd, key, env string) string {
	envValue := os.Getenv(env)
	if envValue != "" {
		cmd += " " + key + " " + envValue
	}
	return cmd
}

func generateArgs() []string {
	// env ENTRY_POINT_CMD has the highest priority
	entryPointCmd := os.Getenv("MOSN_ENTRY_POINT_CMD")
	if entryPointCmd != "" {
		return strings.Fields(mosnCmd + "" + entryPointCmd)
	}

	args := mosnCmd
	args += " -c " + DefaultMosnConfigPath
	args = addCmdArg(args, "--log-level", "LOG_LEVEL")
	args = addCmdArg(args, "--feature-gates", "FEATURE_GATES")
	args = addCmdArg(args, "--component-log-level", "COMPONENT_LOG_LEVEL")
	args = addCmdArg(args, "--local-address-ip-version", "LOCAL_ADDRESS_IP_VERSION")
	args = addCmdArg(args, "--restart-epoch", "RESTART_EPOCH")
	args = addCmdArg(args, "--drain-time-s", "DRAIN_TIME_S")
	args = addCmdArg(args, "--parent-shutdown-time-s", "PARENT_SHUTDOWN_TIME_S")
	args = addCmdArg(args, "--max-obj-name-len", "MAX_OBJ_NAME_LEN")
	args = addCmdArg(args, "--concurrency", "CONCURRENCY")
	args = addCmdArg(args, "--log-format-prefix-with-location", "LOG_FORMAT_PREFIX_WITH_LOCATION")
	args = addCmdArg(args, "--bootstrap-version", "BOOTSTRAP_VERSION")
	args = addCmdArg(args, "--drain-strategy", "DRAIN_STRATEGY")
	args = addCmdArg(args, "--disable-hot-restart", "DISABLE_HOT_RESTART")

	envAppendCmd := os.Getenv("MOSN_APPEND_CMD")
	if envAppendCmd != "" {
		args += envAppendCmd
	}

	return strings.Fields(args)
}

func main() {
}
