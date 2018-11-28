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

var defaultGenerator IDGenerator

// IDGenerator utility to generate auto-increment ids
type IDGenerator struct {
	counter uint64
}

// Get get id
func (g *IDGenerator) Get() uint64 {
	return atomic.AddUint64(&g.counter, 1)
}

// Get get id in string format
func (g *IDGenerator) GetString() string {
	n := atomic.AddUint64(&g.counter, 1)
	return strconv.FormatUint(n, 10)
}

// GenerateID get id by default global generator
func GenerateID() uint64 {
	return defaultGenerator.Get()
}

// GenerateIDString get id string by default global generator
func GenerateIDString() string {
	return defaultGenerator.GetString()
}

// StreamIDConv convert streamID from uint64 to string
func StreamIDConv(streamID uint64) string {
	return strconv.FormatUint(streamID, 10)
}

// RequestIDConv convert streamID from string to uint64
func RequestIDConv(streamID string) uint64 {
	reqID, _ := strconv.ParseUint(streamID, 10, 64)
	return reqID
}
