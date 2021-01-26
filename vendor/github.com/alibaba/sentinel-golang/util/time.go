package util

import (
	"time"
)

const (
	TimeFormat         = "2006-01-02 15:04:05"
	DateFormat         = "2006-01-02"
	UnixTimeUnitOffset = uint64(time.Millisecond / time.Nanosecond)
)

// FormatTimeMillis converts Millisecond to time string
// tsMillis accurates to millisecond，otherwise, an exception will occur
func FormatTimeMillis(tsMillis uint64) string {
	return time.Unix(0, int64(tsMillis*UnixTimeUnitOffset)).Format(TimeFormat)
}

// FormatDate converts to date string
// tsMillis accurates to millisecond，otherwise, an exception will occur
func FormatDate(tsMillis uint64) string {
	return time.Unix(0, int64(tsMillis*UnixTimeUnitOffset)).Format(DateFormat)
}

// Returns the current Unix timestamp in milliseconds.
func CurrentTimeMillis() uint64 {
	return uint64(time.Now().UnixNano()) / UnixTimeUnitOffset
}

// Returns the current Unix timestamp in nanoseconds.
func CurrentTimeNano() uint64 {
	return uint64(time.Now().UnixNano())
}
