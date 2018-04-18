package utility

import (
	"time"
	"strconv"
)

func GenerateExceptionStreamID(reason string) string {
	return "exception-" + reason + "-" + time.Now().String()
}

func GenerateDefaultStreamID() string {
	return "streamID-" + time.Now().String()
}

func StreamIDConvert(reqID uint32)string {
	return strconv.FormatUint(uint64(reqID), 10)
}
