package rogger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

//DEBUG loglevel
const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	OFF
)

var (
	logLevel = DEBUG
	colored  = false

	logQueue      = make(chan *logValue, 10000)
	loggerMap     = make(map[string]*Logger)
	writeDone     = make(chan bool)
	loggermaplock = new(sync.Mutex)
	currUnixTime  int64
	currDateTime  string
	currDateHour  string
	currDateDay   string
)

//Logger is the struct with name and wirter.
type Logger struct {
	name   string
	writer LogWriter
}

//LogLevel is uint8 type
type LogLevel uint8

type logValue struct {
	level  LogLevel
	value  []byte
	fileNo string
	writer LogWriter
}

func init() {
	now := time.Now()
	currUnixTime = now.Unix()
	currDateTime = now.Format("2006-01-02 15:04:05")
	currDateHour = now.Format("2006010215")
	currDateDay = now.Format("20060102")
	go func() {
		tm := time.NewTimer(time.Second)
		if err := recover(); err != nil { // avoid timer panic
		}
		for {
			now := time.Now()
			d := time.Second - time.Duration(now.Nanosecond())
			tm.Reset(d)
			<-tm.C
			now = time.Now()
			currUnixTime = now.Unix()
			currDateTime = now.Format("2006-01-02 15:04:05")
			currDateHour = now.Format("2006010215")
			currDateDay = now.Format("20060102")
		}
	}()
	go flushLog(true)
}

//String return turns the LogLevel to string.
func (lv *LogLevel) String() string {
	switch *lv {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

//Colored enable colored level string when use console writer
func (lv *LogLevel) coloredString() string {
	switch *lv {
	case DEBUG:
		return "\x1b[34mDEBUG\x1b[0m" //blue
	case INFO:
		return "\x1b[32mINFO\x1b[0m" //green
	case WARN:
		return "\x1b[33mWARN\x1b[0m" // yellow
	case ERROR:
		return "\x1b[31mERROR\x1b[0m" //cred
	default:
		return "\x1b[37mUNKNOWN\x1b[0m" // white
	}
}

// GetLogger return an logger instance
func GetLogger(name string) *Logger {
	loggermaplock.Lock()
	defer loggermaplock.Unlock()
	if lg, ok := loggerMap[name]; ok {
		return lg
	}
	lg := &Logger{
		name:   name,
		writer: &ConsoleWriter{},
	}
	loggerMap[name] = lg
	return lg
}

//SetLevel sets the log level
func SetLevel(level LogLevel) {
	logLevel = level
}

//Colored enable colored level string when use console writer
func Colored() {
	colored = true
}

//StringToLevel turns string to LogLevel
func StringToLevel(level string) LogLevel {
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return DEBUG
	}
}

//SetLogName sets the log name
func (l *Logger) SetLogName(name string) {
	l.name = name
}

//SetFileRoller sets the file rolled by size in MB, with max num of files.
func (l *Logger) SetFileRoller(logpath string, num int, sizeMB int) error {
	if err := os.MkdirAll(logpath, 0755); err != nil {
		panic(err)
	}
	w := NewRollFileWriter(logpath, l.name, num, sizeMB)
	l.writer = w
	return nil
}

//IsConsoleWriter returns whether is consoleWriter or not.
func (l *Logger) IsConsoleWriter() bool {
	if reflect.TypeOf(l.writer) == reflect.TypeOf(&ConsoleWriter{}) {
		return true
	}
	return false
}

//SetWriter sets the writer to the logger.
func (l *Logger) SetWriter(w LogWriter) {
	l.writer = w
}

//SetDayRoller sets the logger to rotate by day, with max num files.
func (l *Logger) SetDayRoller(logpath string, num int) error {
	if err := os.MkdirAll(logpath, 0755); err != nil {
		return err
	}
	w := NewDateWriter(logpath, l.name, DAY, num)
	l.writer = w
	return nil
}

//SetHourRoller sets the logger to rotate by hour, with max num files.
func (l *Logger) SetHourRoller(logpath string, num int) error {
	if err := os.MkdirAll(logpath, 0755); err != nil {
		return err
	}
	w := NewDateWriter(logpath, l.name, HOUR, num)
	l.writer = w
	return nil
}

//SetConsole sets the logger with console writer.
func (l *Logger) SetConsole() {
	l.writer = &ConsoleWriter{}
}

//Debug logs interface in debug loglevel.
func (l *Logger) Debug(v ...interface{}) {
	l.writef(DEBUG, "", v)
}

//Info logs interface in Info loglevel.
func (l *Logger) Info(v ...interface{}) {
	l.writef(INFO, "", v)
}

//Warn logs interface in warning loglevel
func (l *Logger) Warn(v ...interface{}) {
	l.writef(WARN, "", v)
}

//Error logs interface in Error loglevel
func (l *Logger) Error(v ...interface{}) {
	l.writef(ERROR, "", v)
}

//Debugf logs interface in debug loglevel with formating string
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.writef(DEBUG, format, v)
}

//Infof logs interface in Infof loglevel with formating string
func (l *Logger) Infof(format string, v ...interface{}) {
	l.writef(INFO, format, v)
}

//Warnf logs interface in warning loglevel with formating string
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.writef(WARN, format, v)
}

//Errorf logs interface in Error loglevel with formating string
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.writef(ERROR, format, v)
}

func (l *Logger) writef(level LogLevel, format string, v []interface{}) {
	if level < logLevel {
		return
	}

	buf := bytes.NewBuffer(nil)
	if l.writer.NeedPrefix() {
		fmt.Fprintf(buf, "%s|", currDateTime)
		if logLevel == DEBUG {
			_, file, line, ok := runtime.Caller(2)
			if !ok {
				file = "???"
				line = 0
			} else {
				file = filepath.Base(file)
			}
			fmt.Fprintf(buf, "%s:%d|", file, line)
		}
		if colored && l.IsConsoleWriter() {
			buf.WriteString(level.coloredString())
		} else {
			buf.WriteString(level.String())
		}
		buf.WriteByte('|')
	}

	if format == "" {
		fmt.Fprint(buf, v...)
	} else {
		fmt.Fprintf(buf, format, v...)
	}
	if l.writer.NeedPrefix() {
		buf.WriteByte('\n')
	}
	logQueue <- &logValue{value: buf.Bytes(), writer: l.writer}
}

func getFuncName(name string) string {
	idx := strings.LastIndexByte(name, '/')
	if idx != -1 {
		name = name[idx:]
		idx = strings.IndexByte(name, '.')
		if idx != -1 {
			name = strings.TrimPrefix(name[idx:], ".")
		}
	}
	return name
}

//FlushLogger flushs all log to disk.
func FlushLogger() {
	flushLog(false)
}

func flushLog(sync bool) {
	if sync {
		for v := range logQueue {
			v.writer.Write(v.value)
		}
	} else {
		for {
			select {
			case v := <-logQueue:
				v.writer.Write(v.value)
				continue
			default:
				return
			}
		}
	}
}
