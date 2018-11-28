package gxsync

import (
	"math/rand"
	"testing"
)

const (
	SEQ_TEST_N   = 100000
	FUZZY_TEST_N = 100000
)

func TestUnbounedChan_Seq(t *testing.T) {
	q := NewUnboundedChan()
	seqTestUnbounedChan(t, q)
}

func TestUnbounedChan_Fuzzy(t *testing.T) {
	q := NewUnboundedChan()
	fuzzyTestUnbounedChan(t, q)
}

func seqTestUnbounedChan(t *testing.T, q *UnboundedChan) {
	vals := []interface{}{}
	for i := 0; i < SEQ_TEST_N; i++ {
		vals = append(vals, rand.Int())
	}

	for i, val := range vals {
		q.Push(val)
		if q.Len() != -1 && q.Len() != i+1 {
			t.Fatalf("queue length should be %v, but is %v", i+1, q.Len())
		}
	}

	for i := 0; i < SEQ_TEST_N; i++ {
		val := q.Pop()
		if val != vals[i] {
			t.Fatalf("pop val should be %v, but is %v", vals[i], val)
		}
		if q.Len() != -1 && q.Len() != SEQ_TEST_N-i-1 {
			t.Fatalf("queue length should be %v, but is %v", SEQ_TEST_N-i-1, q.Len())
		}
	}

	if q.Len() != 0 {
		t.Fatal("not zero")
	}

	q.Close()
	if v := q.Pop(); v != nil {
		t.Fatal("not nil")
	}

	if v, ok := q.TryPop(); v != nil || !ok {
		t.Fatal("TryPop error after close")
	}
}

func fuzzyTestUnbounedChan(t *testing.T, q *UnboundedChan) {
	vals := []interface{}{}

	for i := 0; i < FUZZY_TEST_N; i++ {
		if q.Len() > 0 && rand.Float64() < 0.4 {
			v := q.Pop()
			if v != vals[0] {
				t.Fatalf("pop val should be %v, but is %v", vals[i], v)
			}
			vals = vals[1:]
		} else {
			v := rand.Int()
			vals = append(vals, v)
			q.Push(v)
		}

		if q.Len() != -1 && q.Len() != len(vals) {
			t.Fatalf("queue length should be %v, but is %v", len(vals), q.Len())
		}
	}

	for _, val := range vals {
		pv := q.Pop()
		if val != pv {
			t.Fatalf("pop val should be %v, but is %v", val, pv)
		}
	}
}

type unbounedChanByChannel struct {
	channel chan interface{}
}

func newUnbounedChanByChan() *unbounedChanByChannel {
	ch := &unbounedChanByChannel{
		channel: make(chan interface{}, 1000000),
	}
	return ch
}

func (q *unbounedChanByChannel) Pop() interface{} {
	return <-q.channel
}

func (q *unbounedChanByChannel) TryPop() (interface{}, bool) {
	select {
	case v := <-q.channel:
		return v, true
	default:
		return nil, false
	}
}

func (q *unbounedChanByChannel) Push(v interface{}) {
	q.channel <- v
}

func (q *unbounedChanByChannel) Len() int {
	return -1
}

func (q *unbounedChanByChannel) Close() {
	close(q.channel)
}

func BenchmarkUnbounedChan(b *testing.B) {
	q := NewUnboundedChan()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Push(1)
		q.Pop()
	}
}

func BenchmarkUnbounedChanByChannel(b *testing.B) {
	q := newUnbounedChanByChan()
	for i := 0; i < b.N; i++ {
		q.Push(1)
		q.Pop()
	}
}
