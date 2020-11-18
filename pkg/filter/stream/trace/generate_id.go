package trace

import (
	"encoding/hex"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	TraceIDGeneratorUpperBound uint32 = 8000
	TraceIDGeneratorLowerBound uint32 = 1000
	OSPid                      int
	traceid                    uint32
	LocalIp                    string
)

func init() {
	OSPid = os.Getpid()
	LocalIp, _ = getLocalIp()
}

func GetNextTraceID() uint32 {
	return atomic.AddUint32(&traceid, 1)%TraceIDGeneratorUpperBound + TraceIDGeneratorLowerBound
}

type TraceIDGenerator struct {
	ip    string
	hexip []byte
}

func NewTraceIDGenerator(ip string) (*TraceIDGenerator, error) {
	if len(ip) == 0 || ip == "" {
		ip = LocalIp
	}

	t := &TraceIDGenerator{
		ip: ip,
	}
	if err := t.polyfill(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TraceIDGenerator) polyfill() error {
	ip := net.ParseIP(t.ip)
	if v4 := ip.To4(); v4 != nil {
		t.hexip = append(t.hexip, make([]byte, hex.EncodedLen(len(v4)))...)
		n := hex.Encode(t.hexip, []byte(v4))
		t.hexip = t.hexip[:n]
		return nil
	} else {
		return errors.New("not valid IPv4 address")
	}
}

func (tc *TraceIDGenerator) GetHexIP() []byte {
	return tc.hexip
}

// generate new traceid
func (tc *TraceIDGenerator) AppendBytes(d []byte) []byte {
	d = append(d, tc.hexip...)
	d = strconv.AppendInt(d, unixMilli(), 10)
	d = strconv.AppendInt(d, int64(GetNextTraceID()), 10)
	d = strconv.AppendInt(d, int64(OSPid), 10)
	return d
}

func (tc *TraceIDGenerator) Bytes() []byte {
	return tc.AppendBytes(nil)
}

func (tc *TraceIDGenerator) String() string {
	return string(tc.Bytes())
}

// unixMilli returns a Unix time, the number of milliseconds elapsed since January 1, 1970 UTC.
func unixMilli() int64 {
	scale := int64(time.Millisecond / time.Nanosecond)
	return time.Now().UnixNano() / scale
}

func getLocalIp() (string, error) {
	faces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var addr net.IP
	for _, face := range faces {
		if !isValidNetworkInterface(face) {
			continue
		}

		addrs, err := face.Addrs()
		if err != nil {
			return "", err
		}

		if ipv4, ok := getValidIPv4(addrs); ok {
			addr = ipv4
		}
	}

	if addr == nil {
		return "", errors.New("can not get local IP")
	}

	return addr.String(), nil
}

func getValidIPv4(addrs []net.Addr) (net.IP, bool) {
	for _, addr := range addrs {
		var ip net.IP

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil || ip.IsLoopback() {
			continue
		}

		ip = ip.To4()
		if ip == nil {
			// not an valid ipv4 address
			continue
		}

		return ip, true
	}
	return nil, false
}

func isValidNetworkInterface(face net.Interface) bool {
	if face.Flags&net.FlagUp == 0 {
		// interface down
		return false
	}

	if face.Flags&net.FlagLoopback != 0 {
		// loopback interface
		return false
	}

	if strings.Contains(strings.ToLower(face.Name), "docker") {
		return false
	}

	return true
}
