package types

import (
	"errors"
	"net"
	"reflect"
	"strconv"
	"time"
)

type AccessLog interface {
	Log(reqHeaders map[string]string, respHeaders map[string]string, requestInfo RequestInfo)
}

type AccessLogFilter interface {
	Decide(reqHeaders map[string]string, requestInfo RequestInfo) bool
}

type AccessLogFormatter interface {
	Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo RequestInfo) string
}

const (
	LogStartTime                string = "StartTime"
	LogRequestReceivedDuration  string = "RequestReceivedDuration"
	LogResponseReceivedDuration string = "ResponseReceivedDuration"
	LogBytesSent                string = "BytesSent"
	LogBytesReceived            string = "BytesReceived"
	LogProtocol                 string = "Protocol"
	LogResponseCode             string = "ResponseCode"
	LogDuration                 string = "Duration"
	LogResponseFlag             string = "ResponseFlag"
	LogUpstreamLocalAddress     string = "UpstreamLocalAddress"
	LogDownstreamLocalAddress   string = "DownstreamLocalAddress"
)

const (
	LogNotFoundError string = "Unkown"
)

var (
	ErrParamsNotAdapted = errors.New("The number of params is not adapted.")
)

type FuncMaps map[string]reflect.Value

func (f FuncMaps) Bind(name string, fn interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(name + " is not callable.")
		}
	}()
	v := reflect.ValueOf(fn)
	v.Type().NumIn()
	f[name] = v
	return
}

func (f FuncMaps) Call(name string, params ...interface{}) (result []reflect.Value, err error) {
	if _, ok := f[name]; !ok {
		err = errors.New(name + " does not exist.")
		return
	}
	if len(params) != f[name].Type().NumIn() {
		err = ErrParamsNotAdapted
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	result = f[name].Call(in)
	return
}

func (f FuncMaps) GetValueInString(val interface{}) string {

	switch v := val.(type) {
	case time.Time:
		return v.String()
	case net.Addr:
		return v.String()
	case time.Duration:
		return v.String()
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return LogNotFoundError
	}
}
