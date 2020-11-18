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

package trace

import (
	"context"
	"encoding/json"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

const (
	TracerTraceidArgName    = "tracer_traceid"
	TracerTraceidHeaderName = "SOFA-TraceId"
	TracerRpcidArgName      = "tracer_rpcid"
	TracerRpcidHeaderName   = "SOFA-RpcId"

	DefaultRpcId = "0.1"
	RpcIdChild   = ".1"

	// request url header for mosn on envoy
	pathPrefix = ":path"

	TracerTraceIdMinSize = 25
	TracerTraceIdMaxSize = 40
)

func init() {
	api.RegisterStream(v2.Trace, CreateTraceFilterFactory)
}

// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	trace *v2.StreamTrace
}

// CreateFilterChain for create trace filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewTraceFilter(context, f.trace)

	// Register the runtime hook for the Trace
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

// CreateTraceFilterFactory for create trace filter factory
func CreateTraceFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}

	cfg := v2.StreamTrace{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	trace, err := checkTrace(cfg)
	if err != nil {
		return nil, err
	}

	return &FilterConfigFactory{trace: &trace}, nil
}

func checkTrace(trace v2.StreamTrace) (v2.StreamTrace, error) {
	if trace.Disable {
		return trace, nil
	}

	if trace.TracerTraceidArg == "" {
		trace.TracerTraceidArg = TracerTraceidArgName
	}

	if trace.TracerTraceidHeader == "" {
		trace.TracerTraceidHeader = TracerTraceidHeaderName
	}

	if trace.TracerRpcidHeader == "" {
		trace.TracerRpcidHeader = TracerRpcidHeaderName
	}

	return trace, nil
}
