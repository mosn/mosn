package gxsync

import (
	"testing"
)

// go test -v -run TestBroadcast
func TestBroadcast(t *testing.T) {
	b := NewBroadcaster()

	// case 1 : one reader
	r := b.Listen()

	b.Write("hello")

	if v, _ := r.Read(); v.(string) != "hello" {
		t.Fatal("error string")
	}

	// case 2: two readers
	r1 := b.Listen()

	b.Write(123)

	if v, _ := r.Read(); v.(int) != 123 {
		t.Fatalf("error read value:%#v", v)
	}

	if v, _ := r1.Read(); v.(int) != 123 {
		t.Fatalf("error read value:%#v", v)
	}

	// case 3: close
	b.Close()

	if _, e := r.Read(); e != ErrBroadcastClosed {
		t.Fatalf("read error:%v", e)
	}

	if _, e := r1.Read(); e != ErrBroadcastClosed {
		t.Fatalf("read error:%v", e)
	}

	// cse 4: write after close
	if e := b.Write(123); e.Error() != "panic error:send on closed channel" {
		t.Fatalf("writer error:%v", e)
	}
}

// go test -v -run TestReceiver_DoneRead
func TestReceiver_DoneRead(t *testing.T) {
	b := NewBroadcaster()
	r := b.Listen()
	b.Write(123)

	var v interface{}
	select {
	case n := <-r.ReadChan():
		v, _ = r.ReadDone(n)
	}

	if v.(int) != int(123) {
		t.Fatalf("error read value:%#v", v)
	}
}
