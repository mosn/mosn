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

package faultinject

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	filter.RegisterNetwork(v2.FAULT_INJECT_NETWORK_FILTER, CreateFaultInjectFactory)
}

type faultInjectConfigFactory struct {
	FaultInject *v2.FaultInject
}

func (f *faultInjectConfigFactory) CreateFilterChain(context context.Context, clusterManager types.ClusterManager, callbacks types.NetWorkFilterChainFactoryCallbacks) {
	rf := NewFaultInjector(f.FaultInject)
	callbacks.AddReadFilter(rf)
}

func CreateFaultInjectFactory(conf map[string]interface{}) (types.NetworkFilterChainFactory, error) {
	return &faultInjectConfigFactory{
		FaultInject: config.ParseFaultInjectFilter(conf),
	}, nil
}
