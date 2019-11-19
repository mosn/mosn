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

package utils

import (
	"runtime"
	"testing"
	"time"
)

func BenchmarkLogTimeNow(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		time.Now()
	}
}

func BenchmarkLogTimeFormat(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		time.Now().Format("2006/01/02 15:04:05.000")
	}
}

func BenchmarkLogTime(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for n := 0; n < b.N; n++ {
		CacheTime()
	}
}
