package network

import (
	"math/rand"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type MyEventListener struct{}

func (el *MyEventListener) OnEvent(event types.ConnectionEvent) {}

func randN() int {
	return rand.Intn(1024) + 1
}

func TestAddConnectionEventListener(t *testing.T) {
	logger, err := log.NewLogger("stdout", log.INFO)
	if err != nil {
		t.Fatal(err)
	}

	c := connection{
		logger: logger,
	}

	n := randN()
	for i := 0; i < n; i++ {
		el0 := &MyEventListener{}
		c.AddConnectionEventListener(el0)
	}

	if len(c.connCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddConnectionEventListener(el0)", n, len(c.connCallbacks))
	}
}

func TestAddBytesReadListener(t *testing.T) {
	logger, err := log.NewLogger("stdout", log.INFO)
	if err != nil {
		t.Fatal(err)
	}

	c := connection{
		logger: logger,
	}

	n := randN()
	for i := 0; i < n; i++ {
		fn1 := func(bytesRead uint64) {}
		c.AddBytesReadListener(fn1)
	}

	if len(c.bytesReadCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddBytesReadListener(fn1)", n, len(c.bytesReadCallbacks))
	}
}

func TestAddBytesSendListener(t *testing.T) {
	logger, err := log.NewLogger("stdout", log.INFO)
	if err != nil {
		t.Fatal(err)
	}

	c := connection{
		logger: logger,
	}

	n := randN()
	for i := 0; i < n; i++ {
		fn1 := func(bytesSent uint64) {}
		c.AddBytesSentListener(fn1)
	}

	if len(c.bytesSendCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddBytesSentListener(fn1)", n, len(c.bytesSendCallbacks))
	}
}
