package router

import (
	"strings"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func getHeaderFormatter(value string, append bool) headerFormatter {
	// TODO: variable headers would be support very soon
	if strings.Index(value, "%") != -1 {
		log.DefaultLogger.Warnf("variable headers not support yet, skip")
		return nil
	}
	return &plainHeaderFormatter{
		isAppend:    append,
		staticValue: value,
	}
}

type plainHeaderFormatter struct {
	isAppend      bool
	staticValue string
}

func (f *plainHeaderFormatter) append() bool {
	return f.isAppend
}

func (f *plainHeaderFormatter) format(requestInfo types.RequestInfo) string {
	return f.staticValue
}

