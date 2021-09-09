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
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"mosn.io/mosn/pkg/plugin/proto"
)

var parentPid int

func init() {
	parentPid = os.Getppid()
}

type server struct {
	// This s the real implementation
	Impl Service
}

func (s *server) Call(ctx context.Context, req *proto.Request) (*proto.Response, error) {
	return s.Impl.Call(req)
}

func checkParentAlive() {
	go func() {
		for {
			if parentPid == 1 || os.Getppid() != parentPid {
				fmt.Println("parent no alive, exit")
				os.Exit(0)
			}
			_, err := os.FindProcess(parentPid)
			if err != nil {
				fmt.Println("parent no alive, exit")
				os.Exit(0)
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

// Serve is a function used to serve a plugin. This should be ran on the plugin's main process.
func Serve(service Service) {
	checkParentAlive()

	p := os.Getenv("MOSN_PROCS")
	if procs, err := strconv.Atoi(p); err == nil {
		runtime.GOMAXPROCS(procs)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"MOSN_SERVICE": &Plugin{Impl: service},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
