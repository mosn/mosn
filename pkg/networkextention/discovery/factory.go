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

// ServiceAddrInfo is called by dynamic cluster module for add or update cluster info
type ServiceAddrInfo interface {
	// GetServiceIP returns service ip
	GetServiceIP() string
	// GetServicePort returns service port
	GetServicePort() uint32
	// GetServiceSite returns servie site name
	GetServiceSite() string
}

// Discovery is a service discovery wrapper
type Discovery interface {
	// GetAllServiceName returns
	GetAllServiceName() []string
	// GetServiceAddrInfo returns 'ServiceAddrInfo by the service name 'sn'
	GetServiceAddrInfo(sn string) []ServiceAddrInfo
	// CheckAndResetServiceChange is used to check if the data of 'sn' service has changed, And reset the changed status
	CheckAndResetServiceChange(sn string) bool
	// SetServiceChangeRetry is set the status of 'sn' service changed status to true
	SetServiceChangeRetry(sn string) bool
}

// RegisterDiscovery is used to register discovery implement
func RegisterDiscovery(typ string, discovery Discovery) {
	discoverys.Store(typ, discovery)

}

// GetDiscovery returns the service discovery instance with the name 'typ'
func GetDiscovery(typ string) (Discovery, error) {
	dy, ok := discoverys.Load(typ)
	if !ok {
		// TODO add default discovery
		return nil, errors.New("GetDiscovery failed, not found: " + typ)
	}
	return dy.(Discovery), nil
}
