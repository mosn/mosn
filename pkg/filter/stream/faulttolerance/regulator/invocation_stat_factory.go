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

package regulator

import (
	"sync"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

type InvocationStatFactory struct {
	invocationStats *sync.Map
	regulator       Regulator
}

var invocationStatFactoryInstance *InvocationStatFactory

func GetInvocationStatFactoryInstance() *InvocationStatFactory {
	return invocationStatFactoryInstance
}

func NewInvocationStatFactory(config *v2.FaultToleranceFilterConfig) *InvocationStatFactory {
	invocationStatFactoryInstance = &InvocationStatFactory{
		invocationStats: new(sync.Map),
		regulator:       NewDefaultRegulator(config),
	}
	return invocationStatFactoryInstance
}

func (f *InvocationStatFactory) GetInvocationStat(host api.HostInfo, dimension InvocationDimension) *InvocationStat {
	key := dimension.GetInvocationKey()
	if value, ok := f.invocationStats.Load(key); ok {
		return value.(*InvocationStat)
	} else {
		stat := NewInvocationStat(host, dimension)
		if value, ok := f.invocationStats.LoadOrStore(key, stat); ok {
			return value.(*InvocationStat)
		} else {
			f.regulator.Regulate(stat)
			return value.(*InvocationStat)
		}
	}
}

func (f *InvocationStatFactory) ReleaseInvocationStat(key string) {
	f.invocationStats.Delete(key)
}
