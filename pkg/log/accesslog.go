package log

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"fmt"
)

const (
	// todo: add headers
	DefaultAccessLogFormat = "%PROTOCOL% %RESPONSE_CODE% %RESPONSE_FLAGS% ..."
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

// format example: "{request info field a} {request info field b} {request info field c} {request info field d} ..."
func formatToFormatters(format string) []types.AccessLogFormatter {
	// TODO: format to formatters

	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{},
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
	fieldName string
}

func (f *simpleRequestInfoFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	// todo: map fieldName to field vale string
	return fmt.Sprintf("%+v", requestInfo)
}

type simpleReqHeadersFormatter struct {
}

func (f *simpleReqHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	return "" // todo: reqHeaders
}

type simpleRespHeadersFormatter struct {
}

func (f *simpleRespHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	return "" // todo: respHeaders
}
