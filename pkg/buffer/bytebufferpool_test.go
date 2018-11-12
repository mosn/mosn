package buffer

import (
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func intRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func intN(n int) int {
	return rand.Intn(n) + 1
}

func TestByteBufferPoolSmallBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := intN(1 << minShift)
		ob := pool.take(size)
		// Puts the bytes to pool
		pool.give(ob)

		nb := pool.take(size)
		if nb == ob {
			t.Errorf("Expect get the different bytes from pool, but got %p %p", nb, ob)
		}
	}
}

func TestBytesBufferPoolMediumBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := intRange(1<<minShift+1, 1<<maxShift)
		ob := pool.take(size)
		// Puts the bytes to pool
		pool.give(ob)

		nb := pool.take(size)
		if nb != ob {
			t.Errorf("Expect get the same bytes from pool, but got %p %p", nb, ob)
		}
	}
}

func TestBytesBufferPoolLargeBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := 1<<maxShift + intN(i+1)
		ob := pool.take(size)
		// Puts the bytes to pool
		pool.give(ob)

		nb := pool.take(size)
		if nb == ob {
			t.Errorf("Expect get the different bytes from pool, but got %p %p", nb, ob)
		}
	}
}
