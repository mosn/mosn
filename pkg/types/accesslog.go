package types

import (
	"errors"
	"strconv"
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

func StartTimeGetter(info RequestInfo) string {
	return info.StartTime().String()
}

func ReceivedDurationGetter(info RequestInfo) string {
	return info.RequestReceivedDuration().String()
}

func ResponseReceivedDurationGetter(info RequestInfo) string {
	return info.ResponseReceivedDuration().String()
}

func BytesSentGetter(info RequestInfo) string {
	return strconv.FormatUint(info.BytesSent(), 10)
}

func BytesReceivedGetter(info RequestInfo) string {

	return strconv.FormatUint(info.BytesReceived(), 10)
}

func ProtocolGetter(info RequestInfo) string {
	return string(info.Protocol())
}

func ResponseCodeGetter(info RequestInfo) string {
	return strconv.FormatUint(uint64(info.ResponseCode()),10)
}

func DurationGetter(info RequestInfo) string {
	return info.Duration().String()
}

func GetResponseFlagGetter(info RequestInfo) string {

	return strconv.FormatBool(info.GetResponseFlag(0))
}

func UpstreamLocalAddressGetter(info RequestInfo) string {
	return info.UpstreamLocalAddress().String()
}

func DownstreamLocalAddressGetter(info RequestInfo) string {
	return info.DownstreamLocalAddress().String()
}