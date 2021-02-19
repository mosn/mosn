// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

type Action uint32

const (
	ActionContinue Action = 0
	ActionPause    Action = 1
)

type PeerType uint32

const (
	PeerTypeUnknown PeerType = 0
	PeerTypeLocal   PeerType = 1
	PeerTypeRemote  PeerType = 2
)

type LogLevel uint32

const (
	LogLevelTrace    LogLevel = 0
	LogLevelDebug    LogLevel = 1
	LogLevelInfo     LogLevel = 2
	LogLevelWarn     LogLevel = 3
	LogLevelError    LogLevel = 4
	LogLevelCritical LogLevel = 5
	LogLevelMax      LogLevel = 6
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelCritical:
		return "critical"
	default:
		panic("invalid log level")
	}
}

type Status uint32

const (
	StatusOK              Status = 0
	StatusNotFound        Status = 1
	StatusBadArgument     Status = 2
	StatusEmpty           Status = 7
	StatusCasMismatch     Status = 8
	StatusInternalFailure Status = 10
)

type MapType uint32

const (
	MapTypeHttpRequestHeaders       MapType = 0
	MapTypeHttpRequestTrailers      MapType = 1
	MapTypeHttpResponseHeaders      MapType = 2
	MapTypeHttpResponseTrailers     MapType = 3
	MapTypeHttpCallResponseHeaders  MapType = 6
	MapTypeHttpCallResponseTrailers MapType = 7
)

type BufferType uint32

const (
	BufferTypeHttpRequestBody      BufferType = 0
	BufferTypeHttpResponseBody     BufferType = 1
	BufferTypeDownstreamData       BufferType = 2
	BufferTypeUpstreamData         BufferType = 3
	BufferTypeHttpCallResponseBody BufferType = 4
	BufferTypeGrpcReceiveBuffer    BufferType = 5
	BufferTypeVMConfiguration      BufferType = 6
	BufferTypePluginConfiguration  BufferType = 7
	BufferTypeCallData             BufferType = 8
)

type StreamType uint32

const (
	StreamTypeRequest  StreamType = 0
	StreamTypeResponse StreamType = 1
)

type MetricType uint32

const (
	MetricTypeCounter   = 0
	MetricTypeGauge     = 1
	MetricTypeHistogram = 2
)
