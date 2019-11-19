package context

import (
	"context"
	"math/rand"
	"testing"
)

const testNodeNum = 10

var randomTable [testNodeNum]ContextKey

func init() {
	// init random table per-run for all benchmark scenario, so the performance will not be affected by random functions.
	for i := 0; i < testNodeNum; i++ {
		randomTable[i] = ContextKey(rand.Intn(int(ContextKeyEnd)))
	}
}

func TestSetGet(t *testing.T) {
	expected := "egress"
	ctx := context.Background()

	// set
	ctx = WithValue(ctx, ContextKeyListenerType, expected)

	// get
	value := ctx.Value(ContextKeyListenerType)
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
