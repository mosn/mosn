package log

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strings"
)

var (
	RequestInfoFuncMap map[string]interface{}
	funcs              types.FuncMaps
)

func init() {
	funcs = make(types.FuncMaps, 10)
	RequestInfoFuncMap = map[string]interface{} {
		types.LogStartTime:                types.RequestInfo.StartTime,
		types.LogRequestReceivedDuration:  types.RequestInfo.RequestReceivedDuration,
		types.LogResponseReceivedDuration: types.RequestInfo.ResponseReceivedDuration,
		types.LogBytesSent:                types.RequestInfo.BytesSent,
		types.LogBytesReceived:            types.RequestInfo.BytesReceived,
		types.LogProtocol:                 types.RequestInfo.Protocol,
		types.LogResponseCode:             types.RequestInfo.ResponseCode,
		types.LogDuration:                 types.RequestInfo.Duration,
		types.LogResponseFlag:             types.RequestInfo.GetResponseFlag,
		types.LogUpstreamLocalAddress:     types.RequestInfo.UpstreamLocalAddress,
		types.LogDownstreamLocalAddress:   types.RequestInfo.DownstreamLocalAddress,
	}

	for k, v := range RequestInfoFuncMap {
		if err := funcs.Bind(k, v);err != nil{
			DefaultLogger.Errorf("Bind %s: %s", k, err.Error())
		}
	}
}

const (
	//read docs/access-log-details.md
	DefaultAccessLogFormat = "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %BytesSent%" + " " +
		"%BytesReceived% %PROTOCOL% %ResponseCode% %Duration% %RESPONSE_FLAGS%  %RESPONSE_CODE% %RESPONSE_FLAGS%"
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
		DefaultLogger.Debugf("No ReqInfo Format Keys Input")
		return ""
	}

	format := ""
	for _, key := range f.reqInfoFormat {

		if r, err := funcs.Call(key); err != nil {
			DefaultLogger.Errorf(err.Error())
		} else {
			vString := funcs.GetValueInString(r[0].Interface())
			if vString == types.LogNotFoundError {
				DefaultLogger.Errorf("Convert Error Occurs ")
			} else {
				format = format + vString + " "
			}
		}
	}

	//delete the last " "
	format = format[:len(format)-1]
	return format
}

type simpleReqHeadersFormatter struct {
	reqHeaderFormat []string
}

func (f *simpleReqHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {

	if f.reqHeaderFormat == nil {
		DefaultLogger.Debugf("No ReqHeaders Format Keys Input")
		return ""
	}
	format := ""

	for _, key := range f.reqHeaderFormat {
		if v, ok := reqHeaders[key]; ok {
			format = format + "Req." + v + " "
		} else {
			DefaultLogger.Debugf("Invalid RespHeaders Format Keys %s", key)
		}
	}

	//delete the last " "
	format = format[:len(format)-1]
	return format
}

type simpleRespHeadersFormatter struct {
	respHeaderFormat []string
}

func (f *simpleRespHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	if f.respHeaderFormat == nil {
		DefaultLogger.Debugf("No RespHeaders Format Keys Input")
		return ""
	}

	format := ""
	for _, key := range f.respHeaderFormat {

		if v, ok := respHeaders[key]; ok {
			format = format + "Resp." + v + " "
		} else {
			DefaultLogger.Debugf("Invalid RespHeaders Format Keys:%s", key)
		}
	}

	//delete the last " "
	format = format[:len(format)-1]
	return format
}