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
package network

import (
	"context"

	"github.com/alipay/sofamosn/pkg/api/v2"
	"github.com/alipay/sofamosn/pkg/filter/network/faultinject"
	"github.com/alipay/sofamosn/pkg/filter/network/tcpproxy"
	"github.com/alipay/sofamosn/pkg/types"
)

type FaultInjectFilterConfigFactory struct {
	FaultInject *v2.FaultInject
	Proxy       *v2.TcpProxy
}

func (fifcf *FaultInjectFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager,
	context context.Context) types.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(faultinject.NewFaultInjecter(fifcf.FaultInject))
		manager.AddReadFilter(tcpproxy.NewProxy(fifcf.Proxy, clusterManager, context))
	}
}
