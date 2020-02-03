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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"

	pbtypes "github.com/gogo/protobuf/types"

	"github.com/urfave/cli"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/config"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/mosn"
	"mosn.io/mosn/pkg/types"
)

var (
	cmdStart = cli.Command{
		Name:  "start",
		Usage: "start mosn proxy",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "config, c",
				Usage:  "Load configuration from `FILE`",
				EnvVar: "MOSN_CONFIG",
				Value:  "configs/mosn_config.json",
			}, cli.StringFlag{
				Name:   "service-cluster, s",
				Usage:  "sidecar service cluster",
				EnvVar: "SERVICE_CLUSTER",
			}, cli.StringFlag{
				Name:   "service-node, n",
				Usage:  "sidecar service node",
				EnvVar: "SERVICE_NODE",
			}, cli.StringSliceFlag{
				Name:   "service-meta, sm",
				Usage:  "sidecar service metadata",
				EnvVar: "SERVICE_META",
			}, cli.StringFlag{
				Name:   "feature-gates, f",
				Usage:  "config feature gates",
				EnvVar: "FEATURE_GATES",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			serviceCluster := c.String("service-cluster")
			serviceNode := c.String("service-node")
			serviceMeta := c.StringSlice("service-meta")
			conf := config.Load(configPath)
			// set feature gates
			err := featuregate.Set(c.String("feature-gates"))
			if err != nil {
				log.StartLogger.Infof("[mosn] [start] parse feature-gates flag fail : %+v", err)
				os.Exit(1)
			}
			// start pprof
			if conf.Debug.StartDebug {
				port := 9090 //default use 9090
				if conf.Debug.Port != 0 {
					port = conf.Debug.Port
				}
				addr := fmt.Sprintf("0.0.0.0:%d", port)
				s := &http.Server{Addr: addr, Handler: nil}
				store.AddService(s, "pprof", nil, nil)
			}
			// set mosn metrics flush
			metrics.FlushMosnMetrics = true
			// set version and go version
			metrics.SetVersion(Version)
			metrics.SetGoVersion(runtime.Version())
			initXdsFlags(serviceCluster, serviceNode, serviceMeta)

			mosn.Start(conf)
			return nil
		},
	}

	cmdStop = cli.Command{
		Name:  "stop",
		Usage: "stop mosn proxy",
		Action: func(c *cli.Context) error {
			return nil
		},
	}

	cmdReload = cli.Command{
		Name:  "reload",
		Usage: "reconfiguration",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
)

const serviceMetaSeparator = ":"

func initXdsFlags(serviceCluster, serviceNode string, serviceMeta []string) {
	info := types.GetGlobalXdsInfo()
	info.ServiceCluster = serviceCluster
	info.ServiceNode = serviceNode
	info.Metadata = &pbtypes.Struct{
		Fields: map[string]*pbtypes.Value{},
	}

	for _, keyValue := range serviceMeta {
		keyValueSep := strings.SplitN(keyValue, serviceMetaSeparator, 2)
		if len(keyValueSep) != 2 {
			continue
		}
		key := keyValueSep[0]
		value := keyValueSep[1]

		info.Metadata.Fields[key] = &pbtypes.Value{
			Kind: &pbtypes.Value_StringValue{
				StringValue: value,
			},
		}
	}
}
