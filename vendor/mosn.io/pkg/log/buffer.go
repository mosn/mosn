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

package log

import (
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

// logPool stores buffers for log.
// we use a separate pool to avoid log data impacting others
var logPool buffer.IoBufferPool

// GetLogBuffer returns a LogBuffer from logPool
func GetLogBuffer(size int) LogBuffer {
	return LogBuffer{
		logbuffer: logPool.GetIoBuffer(size),
	}
}

// PutLogBuffer puts a LogBuffer back to logPool
func PutLogBuffer(buf LogBuffer) error {
	return logPool.PutIoBuffer(buf.buffer())
}

// logbuffer is renamed by api.IoBuffer, makes LogBuffer contains an unexported anonymous member
type logbuffer api.IoBuffer

// LogBuffer is an implementation of api.IoBuffer that used in log package, to distinguish it from api.IoBuffer
// nolint
type LogBuffer struct {
	logbuffer
}

// nolint
var _ api.IoBuffer = LogBuffer{}

func (lb LogBuffer) buffer() api.IoBuffer {
	return lb.logbuffer
}
