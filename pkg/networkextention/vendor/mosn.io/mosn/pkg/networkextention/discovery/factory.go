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

package discovery

import (
	"errors"
	"sync"
)

var (
	discoverys = sync.Map{}
)

type ServiceAddrInfo interface {
	GetServiceIP() string
	GetServicePort() uint32
	GetServiceSite() string
}

type Discovery interface {
	GetAllServiceName() []string
	GetServiceAddrInfo(sn string) []ServiceAddrInfo
	CheckAndResetServiceChange(sn string) bool
	SetServiceChangeRetry(sn string) bool
}

func RegisterDiscovery(typ string, discovery Discovery) {
	discoverys.Store(typ, discovery)

}

func GetDiscovery(typ string) (Discovery, error) {
	dy, ok := discoverys.Load(typ)
	if !ok {
		// TODO add default discovery
		return nil, errors.New("GetDiscovery failed, not found: " + typ)
	}
	return dy.(Discovery), nil
}
