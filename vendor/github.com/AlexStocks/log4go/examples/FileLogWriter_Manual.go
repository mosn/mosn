package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

import (
	l4g "github.com/AlexStocks/log4go"
)

const (
	filename = "flw.log"
)

func newLogRecord(lvl l4g.Level, src string, msg string) *l4g.LogRecord {
	return &l4g.LogRecord{
		Level:   lvl,
		Source:  src,
		Created: time.Now(),
		Message: msg,
	}
}

func testLog(json bool) {
	// Get a new logger instance
	log := l4g.NewLogger()

	// Create a default logger that is logging messages of FINE or higher
	log.AddFilter("file", l4g.FINE, l4g.NewFileLogWriter(filename, false, l4g.DEFAULT_LOG_BUFSIZE))
	// log.Close() // 20160921注释此行代码。此处调用Close，会导致下面的log无法输出

	/* Can also specify manually via the following: (these are the defaults) */
	flw := l4g.NewFileLogWriter(filename, false, l4g.DEFAULT_LOG_BUFSIZE)
	flw.SetFormat("[%D %T] [%L] (%S) %M")
	flw.SetJson(json)
	flw.SetRotate(false)
	flw.SetRotateSize(0)
	flw.SetRotateLines(0)
	flw.SetRotateDaily(false)
	log.AddFilter("file", l4g.FINE, flw)
	log.SetAsDefaultLogger()

	// Log some experimental messages
	log.Finest("Everything is created now (notice that I will not be printing to the file)")
	log.Info("The time is now: %s", time.Now().Format("15:04:05 MST 2006/01/02"))
	log.Critical("Time to close out!")
	l4g.Info("l4g Global info")

	lr := newLogRecord(l4g.CRITICAL, "source",
		"/test/group%3Dbjtelecom%26protocol%3Dpb%26role%3DSRT_Provider%26service%3Dshopping%26version%3D1.0.1")
	log.Warn("log record:%#v", lr)

	// Close the log
	log.Close()

	// Print what was logged to the file (yes, I know I'm skipping error checking)
	fd, _ := os.Open(filename)
	in := bufio.NewReader(fd)
	fmt.Print("Messages logged to file were: (line numbers not included)\n")
	for lineno := 1; ; lineno++ {
		line, err := in.ReadString('\n')
		if err == io.EOF {
			break
		}
		fmt.Printf("%3d:\t%s", lineno, line)
	}
	fd.Close()

	// Remove the file so it's not lying around
	os.Remove(filename)
}

func main() {
	testLog(false)
	testLog(true)
}
