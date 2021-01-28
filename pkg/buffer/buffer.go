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

package buffer

import (
	"mosn.io/pkg/buffer"
)

// TempBufferCtx is template for types.BufferPoolCtx
// Deprecated: use mosn.io/pkg/buffer/buffer.go:TempBufferCtx instead
type TempBufferCtx = buffer.TempBufferCtx

// Deprecated: use mosn.io/pkg/buffer/buffer.go:RegisterBuffer instead
var RegisterBuffer = buffer.RegisterBuffer

// bufferValue is buffer pool's Value
// Deprecated: use mosn.io/pkg/buffer/buffer.go:BufferValue instead
type bufferValue = buffer.BufferValue

// NewBufferPoolContext returns a context with bufferValue
// Deprecated: use mosn.io/pkg/buffer/buffer.go:NewBufferPoolContext instead
var NewBufferPoolContext = buffer.NewBufferPoolContext

// TransmitBufferPoolContext copy a context
// Deprecated: use mosn.io/pkg/buffer/buffer.go:TransmitBufferPoolContext instead
var TransmitBufferPoolContext = buffer.TransmitBufferPoolContext

// PoolContext returns bufferValue by context
// Deprecated: use mosn.io/pkg/buffer/buffer.go:PoolContext instead
var PoolContext = buffer.PoolContext
