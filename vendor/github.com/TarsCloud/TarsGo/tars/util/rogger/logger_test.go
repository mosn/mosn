package rogger

import (
	"tars/util/logger"
	"testing"
	"time"
)

//XTestLogger test logger writes.
func XTestLogger(t *testing.T) {
	SetLevel(DEBUG)
	lg := GetLogger("debug")
	// lg.SetConsole()
	// lg.SetFileRoller("./logs", 3, 1)
	// lg.SetDayRoller("./logs", 2)
	// lg.SetHourRoller("./logs", 2)
	bs := make([]byte, 1024)
	longmsg := string(bs)
	for i := 0; i < 10; i++ {
		lg.Debug("debugxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		lg.Info(longmsg)
		lg.Warn("warn")
		lg.Error("ERROR")
		time.Sleep(time.Second)
	}
	time.Sleep(time.Millisecond * 100)

	FlushLogger()
}

//XTestGetLogList test get log list
func XTestGetLogList(t *testing.T) {
	w := NewDateWriter("./logs", "abc", HOUR, 0)
	w.cleanOldLogs()
}

//BenchmarkRogger benchmark rogger writes.
func BenchmarkRogger(b *testing.B) {
	SetLevel(DEBUG)
	bs := make([]byte, 1024)
	longmsg := string(bs)
	lg := GetLogger("rogger")
	lg.SetFileRoller("./logs", 10, 100)
	for i := 0; i < b.N; i++ {
		lg.Debug("debugxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		lg.Info(longmsg)
		lg.Warn("warn")
		lg.Error("ERROR")
	}
	FlushLogger()
}

//BenchmarkOldLogger benchmark old rogger writes.
func BenchmarkOldLogger(b *testing.B) {
	bs := make([]byte, 1024)
	longmsg := string(bs)
	lg := logger.GetLogger()
	lg.SetLevel(logger.DEBUG)
	lg.SetConsole(false)
	lg.SetRollingFile("./logs", "oldlog", 10, 100, logger.MB)
	for i := 0; i < b.N; i++ {
		lg.Debug("debugxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		lg.Info(longmsg)
		lg.Warn("warn")
		lg.Error("ERROR")
	}
}
