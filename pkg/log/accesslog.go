package log

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	//"time"
	"strings"
)

const (
	// todo: add headers
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
	} else {
		format += DefaultAccessLogFormat
	}

	return &accesslogformatter{
		formatters: formatToFormatters(format),
	}
}

// format example: "{request info field a} {request info field b} {request info field c} {request info field d} ..."
func formatToFormatters(format string) []types.AccessLogFormatter {
	// TODO: format to formatters

	//format = "%StartTime% %PROTOCOL% %DownstreamLocalAddress% %UpstreamLocalAddress% %RESPONSE_CODE% %RESPONSE_FLAGS%" +
	//	"%REQ.RequestID% %REQ.Version% %REQ.CmdType%" +

	strArray := strings.Split(format," ")

	//delete %
	for i :=0; i < len(strArray);i++{
		strArray[i] = strArray[i][1:len(strArray[i])-1]
	}

	var reqHeaderArray ,respHeaderArray []string

	// set request headers and response headers
	for _,s := range(strArray){
		if strings.HasPrefix(s,"REQ"){
			reqHeaderArray = append(reqHeaderArray,s)
		} else if strings.HasPrefix(s,"RESP"){
			respHeaderArray = append(respHeaderArray,s)
		}
	}

	//delete REQ.
	if reqHeaderArray != nil {

		for i := 0 ; i < len(reqHeaderArray); i ++{
			reqHeaderArray[i] = reqHeaderArray[i][4:]
		}
	}

	//delete Resp.
	if respHeaderArray != nil {

		for i := 0 ; i < len(respHeaderArray); i ++{
			respHeaderArray[i] = respHeaderArray[i][5:]
		}
	}


	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{},
		&simpleReqHeadersFormatter{},
		&simpleRespHeadersFormatter{},

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
	startTime string
	protocol string
}

func (f *simpleRequestInfoFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	// todo: map fieldName to field vale string

	f.startTime = requestInfo.StartTime().String()

	f.protocol = string(requestInfo.Protocol())



	return fmt.Sprintf("%+v", requestInfo)
}

type simpleReqHeadersFormatter struct {

	reqHeaderArray []string
}

func (f *simpleReqHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	return "" // todo: reqHeaders
}

type simpleRespHeadersFormatter struct {
}

func (f *simpleRespHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	return "" // todo: respHeaders
}
