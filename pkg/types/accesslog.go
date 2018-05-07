package types

import (
	"errors"
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
	ReqHeaderPrefix  string = "REQ."
	RespHeaderPrefix string = "RESP."
)

var (
	ErrParamsNotAdapted = errors.New("The number of params is not adapted.")
)