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

package protocol

import (
	"strconv"
	"sync/atomic"
)

var defaultGenerator IdGenerator

// IdGenerator utility to generate auto-increment ids
type IdGenerator struct {
	counter uint32
}

// Get get id
func (g *IdGenerator) Get() uint32 {
	return atomic.AddUint32(&g.counter, 1)
}

// Get get id in string format
func (g *IdGenerator) GetString() string {
	n := atomic.AddUint32(&g.counter, 1)
	return strconv.FormatUint(uint64(n), 10)
}

// GenerateId get id by default global generator
func GenerateId() uint32 {
	return defaultGenerator.Get()
}

// GenerateIdString get id string by default global generator
func GenerateIdString() string {
	return defaultGenerator.GetString()
}

// StreamIdConv convert streamId from uint32 to string
func StreamIdConv(streamId uint32) string {
	return strconv.FormatUint(uint64(streamId), 10)
}
