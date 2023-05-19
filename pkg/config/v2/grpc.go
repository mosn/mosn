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

package v2

import (
	"encoding/json"
	"time"

	"mosn.io/api"
)

var GrpcDefaultGracefulStopTimeout = time.Second * 30

type GRPC struct {
	GRPCConfig
	// GracefulStopTimeout grpc server graceful stop timeout
	GracefulStopTimeout time.Duration `json:"-"`
}

func (g *GRPC) MarshalJSON() (b []byte, err error) {
	g.GRPCConfig.GracefulStopTimeoutConfig.Duration = g.GracefulStopTimeout
	return json.Marshal(g.GRPCConfig)
}

func (g *GRPC) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &g.GRPCConfig); err != nil {
		return err
	}
	g.GracefulStopTimeout = g.GRPCConfig.GracefulStopTimeoutConfig.Duration
	return nil
}

type GRPCConfig struct {
	// GracefulStopTimeoutConfig grpc server graceful stop timeout
	GracefulStopTimeoutConfig api.DurationConfig `json:"graceful_stop_timeout"`
	// ServerName determines which registered grpc server is used.
	// A server_name should be used only once.
	ServerName string `json:"server_name"`
	// GrpcConfig represents the configuration needed to create
	// a registered grpc server, which can be any types, usually json.
	GrpcConfig json.RawMessage `json:"grpc_config"`
}
