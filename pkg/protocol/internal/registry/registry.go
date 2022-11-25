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

package registry

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// All register function is wrapped by internal package.
// The only way to register it by protocol API

var (
	ConnNewPoolFactories = make(map[api.ProtocolName]types.NewConnPool)
	StreamFactories      = make(map[api.ProtocolName]types.ProtocolStreamFactory)
	HttpMappingFactory   = make(map[api.ProtocolName]api.HTTPMapping)
)

func RegisterNewPoolFactory(protocol api.ProtocolName, factory types.NewConnPool) {
	log.DefaultLogger.Infof("[network] [ register pool factory] register protocol: %s factory", protocol)
	ConnNewPoolFactories[protocol] = factory
}

func RegisterProtocolStreamFactory(name api.ProtocolName, factory types.ProtocolStreamFactory) {
	StreamFactories[name] = factory
}

func RegisterMapping(p api.ProtocolName, m api.HTTPMapping) {
	// some protocol maybe not implement this yet, ignore it.
	if m != nil {
		HttpMappingFactory[p] = m
	}
}

// for test only
func Reset() {
	ConnNewPoolFactories = make(map[api.ProtocolName]types.NewConnPool)
	StreamFactories = make(map[api.ProtocolName]types.ProtocolStreamFactory)
	HttpMappingFactory = make(map[api.ProtocolName]api.HTTPMapping)
}
