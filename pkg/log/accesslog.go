package log

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	//"time"
	"errors"
	"reflect"
	"strings"
)

type Funcs map[string]reflect.Value
var (
	RequestInfoFuncMap map[string]interface{}
	ErrParamsNotAdapted = errors.New("The number of params is not adapted.")
	funcs Funcs
)


func init() {

	funcs =  make(Funcs, 10)
	RequestInfoFuncMap = map[string]interface{}{
		types.LogStartTime: types.RequestInfo.StartTime,
		types.LogProtocol:  types.RequestInfo.Protocol,
		types.LogResponseReceivedDuration:types.RequestInfo.ResponseReceivedDuration,
		types.LogBytesSent: types.RequestInfo.BytesSent,
		types.LogBytesReceived:  types.RequestInfo.BytesReceived,
		types.LogResponseCode:types.RequestInfo.ResponseCode,
		types.LogDuration: types.RequestInfo.Duration,
		types.LogResponseFlag:types.RequestInfo.GetResponseFlag,
		types.LogUpstreamLocalAddress: types.RequestInfo.UpstreamLocalAddress,
		types.LogDownstreamLocalAddress:types.RequestInfo.DownstreamLocalAddress,
	}

	for k, v := range RequestInfoFuncMap {
		err := funcs.Bind(k, v)

		if err != nil {
			fmt.Println("Bind %s: %s", k, err)
		}
	}
}

const (
	//read docs/access-log-details.md
	DefaultAccessLogFormat = "%StartTime% %PROTOCOL% %DownstreamLocalAddress% " +
		"%UpstreamLocalAddress% %RESPONSE_CODE% %RESPONSE_FLAGS%"
)

var accesslogs []*accesslog

// types.AccessLog
type accesslog struct {
	output    string
	filter    types.AccessLogFilter
	formatter types.AccessLogFormatter
	logger    Logger
}

func NewAccessLog(output string, filter types.AccessLogFilter,
	format string) (types.AccessLog, error) {
	var err error
	var logger Logger

	if logger, err = NewLogger(output, 0); err != nil {
		return nil, err
	}

	return &accesslog{
		output:    output,
		filter:    filter,
		formatter: NewAccessLogFormatter(format),
		logger:    logger,
	}, nil
}

func Reopen() error {
	for _, al := range accesslogs {
		if err := al.logger.Reopen(); err != nil {
			return err
		}
	}

	return nil
}

func CloseAll() error {
	for _, al := range accesslogs {
		if err := al.logger.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (l *accesslog) Log(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) {
	if l.filter != nil {
		if !l.filter.Decide(reqHeaders, requestInfo) {
			return
		}
	}

	l.logger.Println(l.formatter.Format(reqHeaders, respHeaders, requestInfo))
}

type accesslogformatter struct {
	formatters []types.AccessLogFormatter
}

func NewAccessLogFormatter(format string) types.AccessLogFormatter {
	if format == "" {
		format = DefaultAccessLogFormat
	}

	return &accesslogformatter{
		formatters: formatToFormatters(format),
	}
}

//format to formatters
// Format Example:
// "%StartTime% %PROTOCOL% %DownstreamLocalAddress%  +
//	"%REQ.RequestID% %REQ.Version% %REQ.CmdType%" +
//  "%RESP.RequestID% %RESP.RespCode% "
func formatToFormatters(format string) []types.AccessLogFormatter {

	strArray := strings.Split(format, " ")

	//delete %
	for i := 0; i < len(strArray); i++ {
		strArray[i] = strArray[i][1 : len(strArray[i])-1]
	}

	//classify keys
	var reqInfoArray, reqHeaderArray, respHeaderArray []string
	for _, s := range strArray {
		if strings.HasPrefix(s, "REQ") {
			reqHeaderArray = append(reqHeaderArray, s)

		} else if strings.HasPrefix(s, "RESP") {
			respHeaderArray = append(respHeaderArray, s)
		} else {
			reqInfoArray = append(reqInfoArray, s)
		}
	}

	//delete REQ.
	if reqHeaderArray != nil {
		for i := 0; i < len(reqHeaderArray); i++ {
			reqHeaderArray[i] = reqHeaderArray[i][4:]
		}
	}

	//delete Resp.
	if respHeaderArray != nil {
		for i := 0; i < len(respHeaderArray); i++ {
			respHeaderArray[i] = respHeaderArray[i][5:]
		}
	}

	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{reqInfoFormat: reqInfoArray},
		&simpleReqHeadersFormatter{reqHeaderFormat: respHeaderArray},
		&simpleRespHeadersFormatter{respHeaderFormat: respHeaderArray},
	}
}

func (f *accesslogformatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	var log string

	for _, formatter := range f.formatters {
		log += formatter.Format(reqHeaders, respHeaders, requestInfo)
	}

	return log
}

type simpleRequestInfoFormatter struct {
	reqInfoFormat []string
}

func (f *simpleRequestInfoFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	// todo: map fieldName to field vale string

	if f.reqInfoFormat == nil {
		return ""
	}

	format := ""
	for _,key := range(f.reqInfoFormat){

		r ,_ := funcs.Call(key)
		format = format + r[0].String() + " "
	}

	return format
}

type simpleReqHeadersFormatter struct {
	reqHeaderFormat []string
}

func (f *simpleReqHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {

	if f.reqHeaderFormat == nil {
		return ""
	}

	format := ""
	for _,key := range(f.reqHeaderFormat){

		if v,ok :=reqHeaders[key];ok{

			format = format + "Req."+v + " "
		}else {

		}
	}

	return format
}

type simpleRespHeadersFormatter struct {
	respHeaderFormat []string
}

func (f *simpleRespHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	if f.respHeaderFormat == nil {
		return ""
	}

	format := ""
	for _,key := range(f.respHeaderFormat){

		if v,ok :=respHeaders[key];ok{

			format = format + "Resp."+v + " "
		}else {

		}
	}
	return format
}

func (f Funcs) Bind(name string, fn interface{}) (err error) {
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

func (f Funcs) Call(name string, params ...interface{}) (result []reflect.Value, err error) {
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

