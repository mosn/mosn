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

package network

import (
	"testing"
	"time"
)

func TestPooledTimer(t *testing.T) {
	start := time.Now()
	t1 := acquireTimer(time.Second)
	<-t1.C
	// should less than 1/100s
	if time.Since(start).Seconds()-1 > 0.01 {
		t.Fail()
	}

	releaseTimer(t1)

	start2 := time.Now()
	t2 := acquireTimer(time.Second * 2)
	<-t2.C
	// should less than 1/100s
	if time.Since(start2).Seconds()-2 > 0.01 {
		t.Fail()
	}

	releaseTimer(t2)
}

func BenchmarkPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tm := acquireTimer(time.Microsecond)
		<-tm.C
		releaseTimer(tm)
	}
	b.StopTimer()
	b.ReportAllocs()
}

func BenchmarkStd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tm := time.NewTimer(time.Microsecond)
		<-tm.C
	}
	b.StopTimer()
	b.ReportAllocs()
}
