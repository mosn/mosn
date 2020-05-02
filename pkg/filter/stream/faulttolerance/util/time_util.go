package util

import (
	"time"
)

func GetNowMS() int64 {
	return time.Now().UnixNano() / 1e6
}
