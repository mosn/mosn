// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	FORMAT_DEFAULT = "[%D %T] [%L] (%S) %M" // [时:分:秒 年月日] [level] (class:file:line) message
	FORMAT_SHORT   = "[%t %d] [%L] %M"      // [小时:分钟 年月日] [level] message
	FORMAT_ABBREV  = "[%L] %M"              // [level] message
)

type formatCacheType struct {
	LastUpdateSeconds    int64
	shortTime, shortDate string
	longTime, longDate   string
}

var (
	formatCache   = &formatCacheType{}
	muFormatCache = sync.Mutex{}
	newLine       = "\n"
	logProcessID  = strconv.Itoa(os.Getpid())
)

func init() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
}

var (
	// pool used to format log
	bufPool sync.Pool
)

func newBuf() *bytes.Buffer {
	if v := bufPool.Get(); v != nil {
		return v.(*bytes.Buffer)
	}

	// for a log, 4K should be enough for most case
	// bytes.Buffer will reallocate a new []byte if previous is run out.
	// when reallocation occurred, the old []byte can only be freed by GC
	return bytes.NewBuffer(make([]byte, 0, 4096))
}

func putBuf(bb *bytes.Buffer) {
	bb.Reset()
	bufPool.Put(bb)
}

func setFormatCache(f *formatCacheType) {
	muFormatCache.Lock()
	defer muFormatCache.Unlock()
	formatCache = f
}

func getFormatCache() *formatCacheType {
	muFormatCache.Lock()
	defer muFormatCache.Unlock()
	return formatCache
}

// Known format codes:
// %T - Time (15:04:05 MST)
// %t - Time (15:04)
// %D - Date (2006/01/02)
// %d - Date (01/02/06)
// %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
// %P - Pid of process
// %S - Source
// %M - Message
// Ignores unknown formats
// Recommended: "[%D %T] [%L] (%S) %M"
func FormatLogRecord(format string, rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}
	if len(format) == 0 {
		return ""
	}

	//out := bytes.NewBuffer(make([]byte, 0, 64))
	out := newBuf()
	defer putBuf(out)

	secs := rec.Created.UnixNano() / 1e9

	cache := getFormatCache()
	// 如果当前msg的时间与上一条消息的时间相等，则直接用cache中的时间string，减小format时间
	if cache.LastUpdateSeconds != secs {
		month, day, year := rec.Created.Month(), rec.Created.Day(), rec.Created.Year()
		hour, minute, second := rec.Created.Hour(), rec.Created.Minute(), rec.Created.Second()
		zone, _ := rec.Created.Zone()
		updated := &formatCacheType{
			LastUpdateSeconds: secs,
			shortTime:         fmt.Sprintf("%02d:%02d", hour, minute),
			shortDate:         fmt.Sprintf("%02d/%02d/%02d", day, month, year%100),
			longTime:          fmt.Sprintf("%02d:%02d:%02d %s", hour, minute, second, zone),
			longDate:          fmt.Sprintf("%04d/%02d/%02d", year, month, day),
		}
		cache = updated
		setFormatCache(updated)
	}

	// Split the string into pieces by % signs
	pieces := bytes.Split([]byte(format), []byte{'%'})

	// Iterate over the pieces, replacing known formats
	sourceFlag := true
	for i, piece := range pieces {
		if i > 0 && len(piece) > 0 {
			switch piece[0] {
			case 'T':
				out.WriteString(cache.longTime)
			case 't':
				out.WriteString(cache.shortTime)
			case 'D':
				out.WriteString(cache.longDate)
			case 'd':
				out.WriteString(cache.shortDate)
			case 'L':
				out.WriteString(levelStrings[rec.Level])
			case 'P':
				out.WriteString(logProcessID)
			case 'S':
				if len(rec.Source) != 0 {
					out.WriteString(rec.Source)
				} else {
					out.UnreadByte()
					out.Truncate(out.Len() - 1)
					sourceFlag = false
				}
			case 's':
				slice := strings.Split(rec.Source, "/")
				out.WriteString(slice[len(slice)-1])
			case 'M':
				out.WriteString(rec.Message)
			}
			if len(piece) > 1 && sourceFlag {
				out.Write(piece[1:])
			}
		} else if len(piece) > 0 {
			out.Write(piece)
		}
	}
	//out.WriteByte('\n')
	out.WriteString(newLine)

	return out.String()
}

// This is the standard writer that prints to standard output.
type FormatLogWriter struct {
	caller bool
	rec    chan LogRecord
	sync.Once
}

// This creates a new FormatLogWriter
func NewFormatLogWriter(out io.Writer, format string) *FormatLogWriter {
	var w = &FormatLogWriter{caller: true}
	w.rec = make(chan LogRecord, LogBufferLength)
	go w.run(out, format)
	return w
}

func (w *FormatLogWriter) run(out io.Writer, format string) {
	for rec := range w.rec {
		if !w.caller {
			rec.Source = ""
		}

		fmt.Fprint(out, FormatLogRecord(format, &rec))
	}
}

// This func shows whether output filename/function/lineno info in log
func (w *FormatLogWriter) GetCallerFlag() bool { return w.caller }

// Set whether output the filename/function name/line number info or not.
// Must be called before the first log message is written.
func (w *FormatLogWriter) SetCallerFlag(flag bool) *FormatLogWriter {
	w.caller = flag
	return w
}

// This is the FormatLogWriter's output method.  This will block if the output
// buffer is full.
func (w *FormatLogWriter) LogWrite(rec *LogRecord) {
	defer func() {
		if e := recover(); e != nil {
			js, err := json.Marshal(rec)
			if err != nil {
				fmt.Printf("json.Marshal(rec:%#v) = error{%#v}\n", rec, err)
				return
			}
			fmt.Printf("log channel has been closed. " + string(js) + "\n")
		}
	}()

	w.rec <- *rec
}

// Close stops the logger from sending messages to standard output.  Attempts to
// send log messages to this logger after a Close have undefined behavior.
func (w *FormatLogWriter) Close() {
	w.Once.Do(func() {
		close(w.rec)
	})
}
