package gxsync

import (
	"testing"
)

import (
	"github.com/cheekybits/is"
	"time"
)

func TestOnceReset(t *testing.T) {
	is := is.New(t)
	var calls int
	var c Once
	go c.Do(func() {
		calls++
	})
	go c.Do(func() {
		calls++
	})
	go c.Do(func() {
		calls++
	})
	time.Sleep(1e9)
	is.Equal(calls, 1)
	c.Reset()
	go c.Do(func() {
		calls++
	})
	go c.Do(func() {
		calls++
	})
	go c.Do(func() {
		calls++
	})
	time.Sleep(1e9)
	is.Equal(calls, 2)
}
