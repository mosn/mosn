package cluster

import (
	"sync"
	"runtime"
	"bytes"
	"fmt"
	"strconv"
	"errors"
)

type Values map[interface{}]interface{}

type golocalstore struct {
	dataLock sync.RWMutex
	data     map[uint64]Values
}

func newgolocalstore() *golocalstore {
	return &golocalstore{
		data:     make(map[uint64]Values),
	}
}

func (g *golocalstore) SetValues(values Values) {
	gid := curGoroutineID()
	lock := g.dataLock

	lock.Lock()
	g.data[gid] = values
	lock.Unlock()
}

func (g *golocalstore) Set(key string, value interface{}) {
	gid := curGoroutineID()
	lock := g.dataLock

	lock.Lock()

	if g.data[gid] == nil {
		g.data[gid] = Values{}
	}

	g.data[gid][key] = value
	lock.Unlock()
}

func (g *golocalstore) Get(key string) interface{} {
	gid := curGoroutineID()
	lock := g.dataLock

	lock.RLock()

	if g.data[gid] == nil {
		lock.RUnlock()
		return nil
	}

	value := g.data[gid][key]
	lock.RUnlock()

	return value
}

var goroutineSpace = []byte("goroutine ")
var littleBuf = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64)
		return &buf
	},
}

func curGoroutineID() uint64 {
	bp := littleBuf.Get().(*[]byte)
	defer littleBuf.Put(bp)
	b := *bp
	b = b[:runtime.Stack(b, false)]
	// Parse the 4707 out of "goroutine 4707 ["
	b = bytes.TrimPrefix(b, goroutineSpace)
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		panic(fmt.Sprintf("No space found in %q", b))
	}
	b = b[:i]
	n, err := parseUintBytes(b, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse goroutine ID out of %q: %v", b, err))
	}
	return n
}

func parseUintBytes(s []byte, base int, bitSize int) (n uint64, err error) {
	var cutoff, maxVal uint64

	if bitSize == 0 {
		bitSize = int(strconv.IntSize)
	}

	s0 := s
	switch {
	case len(s) < 1:
		err = strconv.ErrSyntax
		return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}

	case 2 <= base && base <= 36:
		// valid base; nothing to do

	case base == 0:
		// Look for octal, hex prefix.
		switch {
		case s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X'):
			base = 16
			s = s[2:]
			if len(s) < 1 {
				err = strconv.ErrSyntax
				return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
			}
		case s[0] == '0':
			base = 8
		default:
			base = 10
		}

	default:
		err = errors.New("invalid base " + strconv.Itoa(base))
		return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
	}

	n = 0
	cutoff = cutoff64(base)
	maxVal = 1<<uint(bitSize) - 1

	for i := 0; i < len(s); i++ {
		var v byte
		d := s[i]
		switch {
		case '0' <= d && d <= '9':
			v = d - '0'
		case 'a' <= d && d <= 'z':
			v = d - 'a' + 10
		case 'A' <= d && d <= 'Z':
			v = d - 'A' + 10
		default:
			n = 0
			err = strconv.ErrSyntax
			return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
		}
		if int(v) >= base {
			n = 0
			err = strconv.ErrSyntax
			return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
		}

		if n >= cutoff {
			// n*base overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
		}
		n *= uint64(base)

		n1 := n + uint64(v)
		if n1 < n || n1 > maxVal {
			// n+v overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
		}
		n = n1
	}

	return n, nil
}

// Return the first number n such that n*base >= 1<<64.
func cutoff64(base int) uint64 {
	if base < 2 {
		return 0
	}
	return (1<<64-1)/uint64(base) + 1
}
