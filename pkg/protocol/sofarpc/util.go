package sofarpc

import (
	"strconv"
	"time"
)

func GenerateExceptionStreamID(reason string) string {
	return "exception-" + reason + "-" + time.Now().String()
}

func StreamIDConvert(reqID uint32) string {
	return strconv.FormatUint(uint64(reqID), 10)
}
