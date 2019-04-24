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

package context

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
	"math/rand"
	"testing"
)

const testNodeNum = 10

var randomTable [testNodeNum]types.ContextKey

func init() {
	// init random table per-run for all benchmark scenario, so the performance will not be affected by random functions.
	for i := 0; i < testNodeNum; i++ {
		randomTable[i] = types.ContextKey(rand.Intn(int(types.ContextKeyEnd)))
	}
}

func TestSetGet(t *testing.T) {
	expected := "egress"
	ctx := context.Background()

	// set
	ctx = WithValue(ctx, types.ContextKeyListenerType, expected)

	// get
	value := ctx.Value(types.ContextKeyListenerType)
	if listenerType, ok := value.(string); ok {
		if listenerType != expected {
			t.Errorf("get value error, expected %s, real %s", expected, listenerType)
		}
	} else {
		t.Error("get value type error")
	}

}

func BenchmarkCompatibleGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			ctx.Value(randomTable[i])
		}
	}

}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			Get(ctx, randomTable[i])
		}
	}

}

func BenchmarkSet(b *testing.B) {
	// based on 10 k-v

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		for i := 0; i < testNodeNum; i++ {
			ctx = WithValue(ctx, randomTable[i], struct{}{})
		}
	}
}

func BenchmarkRawGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = context.WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			ctx.Value(randomTable[i])
		}
	}
}

func BenchmarkRawSet(b *testing.B) {
	// based on 10 k-v
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		for i := 0; i < testNodeNum; i++ {
			ctx = context.WithValue(ctx, randomTable[i], struct{}{})
		}

	}
}
