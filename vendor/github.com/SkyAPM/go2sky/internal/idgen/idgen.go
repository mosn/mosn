// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package idgen

import (
	"math/rand"
	"sync"
	"time"
)

var (
	seededIDGen  = rand.New(rand.NewSource(time.Now().UnixNano()))
	seededIDLock sync.Mutex
)

func generateID() int64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return seededIDGen.Int63()
}

// GenerateGlobalID generates global unique id
func GenerateGlobalID() []int64 {
	return []int64{
		time.Now().UnixNano(),
		0,
		generateID(),
	}
}

// GenerateScopedGlobalID generates global unique id with a scopeId prefix
func GenerateScopedGlobalID(scopeID int64) []int64 {
	return []int64{
		scopeID,
		time.Now().UnixNano(),
		generateID(),
	}
}
