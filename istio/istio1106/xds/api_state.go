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

package xds

import (
	"sync"
)

type responseInfo struct {
	ResponseNonce string
	VersionInfo   string
	ResourceNames []string
}

type apiState struct {
	mutex     sync.Mutex
	responses map[string]*responseInfo
}

func newApiState() *apiState {
	return &apiState{
		responses: map[string]*responseInfo{},
	}
}

func (s *apiState) Store(typeUrl string, info *responseInfo) {
	s.mutex.Lock()
	s.responses[typeUrl] = info
	s.mutex.Unlock()
}

func (s *apiState) SetResourceNames(typeUrl string, ResourceNames []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if ok {
		info.ResourceNames = ResourceNames
		s.responses[typeUrl] = info
		return
	}
	s.responses[typeUrl] = &responseInfo{
		ResourceNames: ResourceNames,
	}
}

func (s *apiState) Find(typeUrl string) *responseInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if !ok {
		return &responseInfo{
			ResourceNames: []string{},
		}
	}
	return info
}
