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

package trace

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var traceIdGenerator = newIdGenerator()
var spanIdGeneratorMap = sync.Map{}

type IdGenerator struct {
	index int64
	pid   string
	hexIp string
}

type SpanKey struct {
	TraceId string
	SpanId  string
}

func (spanKey *SpanKey) UniqueKey() string {
	return spanKey.TraceId + "#" + spanKey.SpanId
}

// A span ID generator that generate a span ID like 0.1, 0.1.1, 0.1.2 etc.
type SpanIdGenerator struct {
	key        SpanKey // A combination of traceId and spanId
	childIndex int64
}

func (generator *SpanIdGenerator) GenerateNextChildIndex() string {
	atomic.AddInt64(&generator.childIndex, 1)
	return generator.key.SpanId + "." + strconv.FormatInt(generator.childIndex, 10)
}

func IdGen() IdGenerator {
	return *traceIdGenerator
}

func NewSpanIdGenerator(traceId, spanId string) *SpanIdGenerator {
	return &SpanIdGenerator{
		key: SpanKey{
			TraceId: traceId,
			SpanId:  spanId,
		},
		childIndex: 1,
	}
}

func AddSpanIdGenerator(generator *SpanIdGenerator) {
	spanIdGeneratorMap.Store(generator.key.UniqueKey(), generator)
}

func GetSpanIdGenerator(key *SpanKey) *SpanIdGenerator {
	value, ok := spanIdGeneratorMap.Load(key.UniqueKey())
	if ok {
		return value.(*SpanIdGenerator)
	} else {
		return nil
	}
}

func DeleteSpanIdGenerator(key *SpanKey) {
	spanIdGeneratorMap.Delete(key.UniqueKey())
}

func newIdGenerator() *IdGenerator {
	hexIp := ipToHexString(GetIp())
	return &IdGenerator{
		index: 1000,
		pid:   strconv.Itoa(os.Getpid()),
		hexIp: hexIp,
	}
}

var index int64 = 1000

func (idGen IdGenerator) GenerateTraceId() string {
	return idGen.hexIp + idGen.timeStamp() + idGen.nextId() + idGen.pid
}

func ipToHexString(ip string) string {
	segments := strings.Split(ip, ".")
	result := ""
	for _, segment := range segments {
		i, _ := strconv.Atoi(segment)
		hexString := fmt.Sprintf("%02x", i)
		result += hexString
	}

	return result
}

func GetIp() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		//TODO
	}
	for _, networkInterface := range interfaces {
		addresses, err := networkInterface.Addrs()
		if err != nil {
			//TODO
		}
		for _, address := range addresses {
			switch v := address.(type) {
			case *net.IPNet:
				if !v.IP.IsLoopback() {
					ip := v.IP.To4()
					if ip != nil {
						return ip.String()
					}
				}
			}
		}
	}
	return ""
}
func (idGen IdGenerator) timeStamp() string {
	return strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
}

func (idGen IdGenerator) nextId() string {
	for {
		var current = index
		var next = current + 1
		if current > 9000 {
			next = 1000
		}
		if atomic.CompareAndSwapInt64(&index, current, next) {
			return strconv.FormatInt(index, 10)
		}
	}
}
