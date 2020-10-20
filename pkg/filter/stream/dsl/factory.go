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

package dsl

import (
	"context"
	"encoding/json"

	"mosn.io/api"
	"mosn.io/mosn/pkg/cel"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/extract"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func init() {
	api.RegisterStream(v2.DSL, CreateDSLFilterFactory)
}

var compiler = cel.NewExpressionBuilder(extract.Attributemanifest, cel.CompatCEXL)

type DSL struct {
	BeforeRouterDSL  attribute.Expression
	AfterRouterDSL   attribute.Expression
	AfterBalancerDSL attribute.Expression
	SendFilterDSL    attribute.Expression
	LogDSL           attribute.Expression
}

// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	debug bool
	dsl   *DSL
}

// CreateFilterChain for create dsl filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewDSLFilter(context, f.dsl)

	// Register the runtime hook for the DSL
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	callbacks.AddStreamReceiverFilter(filter, api.AfterChooseHost)
	callbacks.AddStreamSenderFilter(filter)
	callbacks.AddStreamAccessLog(filter)
}

// CreateDSLFilterFactory for create dsl filter factory
func CreateDSLFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}
	cfg := v2.StreamDSL{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	dsl, err := checkDSL(cfg)
	if err != nil {
		return nil, err
	}

	return &FilterConfigFactory{debug: cfg.Debug, dsl: dsl}, nil
}

func checkDSL(cfg v2.StreamDSL) (*DSL, error) {

	dsl := &DSL{}
	if len(cfg.BeforeRouterDSL) != 0 {
		expr, _, err := compiler.Compile(cfg.BeforeRouterDSL)
		if err != nil {
			return nil, err
		}
		dsl.BeforeRouterDSL = expr
	}

	if len(cfg.AfterRouterDSL) != 0 {
		expr, _, err := compiler.Compile(cfg.AfterRouterDSL)
		if err != nil {
			return nil, err
		}
		dsl.AfterRouterDSL = expr
	}

	if len(cfg.AfterBalancerDSL) != 0 {
		expr, _, err := compiler.Compile(cfg.AfterBalancerDSL)
		if err != nil {
			return nil, err
		}
		dsl.AfterBalancerDSL = expr
	}

	if len(cfg.SendFilterDSL) != 0 {
		expr, _, err := compiler.Compile(cfg.SendFilterDSL)
		if err != nil {
			return nil, err
		}
		dsl.SendFilterDSL = expr
	}

	if len(cfg.LogDSL) != 0 {
		expr, _, err := compiler.Compile(cfg.LogDSL)
		if err != nil {
			return nil, err
		}
		dsl.LogDSL = expr
	}

	return dsl, nil
}
