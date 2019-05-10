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
		bp := pool.take(size)

		if cap(*bp) != 1<<minShift {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolMediumBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := minShift; i < maxShift; i++ {
		size := intRange((1<<uint(i))+1, 1<<uint(i+1))
		bp := pool.take(size)

		if cap(*bp) != 1<<uint(i+1) {
			t.Errorf("Expect get the slab size (%d) from pool, but got %d", 1<<uint(i+1), cap(*bp))
		}

		//Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolLargeBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := 1<<maxShift + intN(i+1)
		bp := pool.take(size)

		if cap(*bp) != size {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesSlot(t *testing.T) {
	pool := newByteBufferPool()

	if pool.slot(pool.minSize-1) != 0 {
		t.Errorf("Expect get the 0 slot")
	}

	if pool.slot(pool.minSize) != 0 {
		t.Errorf("Expect get the 0 slot")
	}

	if pool.slot(pool.minSize+1) != 1 {
		t.Errorf("Expect get the 1 slot")
	}

	if pool.slot(pool.maxSize-1) != maxShift-minShift {
		t.Errorf("Expect get the %d slot", maxShift-minShift)
	}

	if pool.slot(pool.maxSize) != maxShift-minShift {
		t.Errorf("Expect get the %d slot", maxShift-minShift)
	}

	if pool.slot(pool.maxSize+1) != errSlot {
		t.Errorf("Expect get errSlot")
	}
}
