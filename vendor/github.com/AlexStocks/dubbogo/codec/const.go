// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

//////////////////////////////////////////
// transport network
//////////////////////////////////////////

type TransportType int

const (
	TRANSPORTTYPE_BEGIN TransportType = iota
	TRANSPORT_TCP
	TRANSPORT_HTTP
	TRANSPORTTYPE_UNKNOWN
)

var transportTypeStrings = [...]string{
	"",
	"TCP",
	"HTTP",
	"",
}

func (t TransportType) String() string {
	if TRANSPORTTYPE_BEGIN < t && t < TRANSPORTTYPE_UNKNOWN {
		return transportTypeStrings[t]
	}

	return ""
}

func GetTransportType(t string) TransportType {
	var typ = TRANSPORTTYPE_UNKNOWN

	for i := TRANSPORTTYPE_BEGIN + 1; i < TRANSPORTTYPE_UNKNOWN; i++ {
		if transportTypeStrings[i] == t {
			typ = TransportType(i)
			break
		}
	}

	return typ
}

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	CODECTYPE_BEGIN CodecType = iota
	CODECTYPE_JSONRPC
	CODECTYPE_DUBBO
	CODECTYPE_UNKNOWN
)

var codecTypeStrings = [...]string{
	"",
	"jsonrpc",
	"dubbo",
	"",
}

func (c CodecType) String() string {
	if CODECTYPE_BEGIN < c && c < CODECTYPE_UNKNOWN {
		return codecTypeStrings[c]
	}

	return ""
}

func GetCodecType(t string) CodecType {
	var typ = CODECTYPE_UNKNOWN

	for i := CODECTYPE_BEGIN + 1; i < CODECTYPE_UNKNOWN; i++ {
		if codecTypeStrings[i] == t {
			typ = CodecType(i)
			break
		}
	}

	return typ
}
