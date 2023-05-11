package util

import (
	"math/rand"
	"sync"
	"time"
)

// tools
var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}
func RandomDuration(min, max time.Duration) time.Duration {
	if min > max {
		return min
	}
	d := r.Int63n(int64(max - min))
	return time.Duration(d) * time.Nanosecond
}
func IsMapEmpty(m *sync.Map) bool {
	empty := true
	//If there is a key in the map, return not empty
	m.Range(func(key, value interface{}) bool {
		empty = false
		return false
	})
	return empty
}
func WaitMapEmpty(m *sync.Map, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				return
			default:
				if IsMapEmpty(m) {
					close(ch)
					return
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}() //check goroutine
	select {
	case <-time.After(timeout):
		close(ch)            // finish check goroutine
		return IsMapEmpty(m) // timeout, retry again
	case <-ch:
		return true //map empty
	}
}
