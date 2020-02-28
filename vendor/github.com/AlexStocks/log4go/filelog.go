// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RotType int32

const (
	ROT_TYPE_DAY  = RotType(1)
	ROT_TYPE_HOUR = RotType(2)
	// rotate file suffix
	DEF_ROT_TIME_SUFFIX_DAY  = "2006-01-02"
	DEF_ROT_TIME_SUFFIX_HOUR = "2006-01-02_15"
)

// This log writer sends output to a file
type FileLogWriter struct {
	LogCloser

	rec chan LogRecord
	rot chan bool

	// The opened file
	filename string
	file     io.WriteCloser
	//file     *os.File

	// The logging format
	json   bool
	format string

	// File header/trailer
	header, trailer string

	// Rotate
	rotate    bool
	rotSuffix string

	// Rotate at linecount
	// 这两个条件与 hourly/daily 可同时有效，然hourly 和 daily 不能同时有效
	maxlines int
	curlines int

	// Rotate at size
	maxsize int64
	cursize int64

	// Rotate daily
	daily bool
	// Rotate hourly
	hourly      bool
	rotInterval int

	rotNextTime int64

	// Keep old logfiles (.001, .002, etc)
	maxbackup int

	caller bool

	sync.Once
}

// This is the FileLogWriter's output method
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	defer func() {
		if e := recover(); e != nil {
			//js, err := json.Marshal(rec)
			//if err != nil {
			//	fmt.Printf("json.Marshal(rec:%#v) = error{%#v}\n", rec, err)
			//	return
			//}
			fmt.Printf("file log channel has been closed. rec:" + String(rec.JSON()) + "\n")
		}
	}()

	w.rec <- *rec
}

func (w *FileLogWriter) Close() {
	w.Once.Do(func() {
		// cost cpu too busy this way, 20190915
		// Wait write coroutine
		// for len(w.rec) > 0 {
		// 	time.Sleep(100 * time.Millisecond)
		// }
		w.WaitClosed(w.rec)

		close(w.rec)

		switch f := w.file.(type) {
		case *os.File:
			f.Sync()
		case *BufFileWriter:
			f.Flush()
		default:
			w.file.Close()
		}
	})
}

// This func shows whether output filename/function/lineno info in log
func (w *FileLogWriter) GetCallerFlag() bool { return w.caller }

func (w *FileLogWriter) checkRotate() bool {
	if !w.rotate {
		return false
	}

	if (w.maxlines > 0 && w.curlines >= w.maxlines) ||
		(w.maxsize > 0 && w.cursize >= w.maxsize) {
		return true
	}

	now := time.Now()
	if w.hourly && now.Unix() >= w.rotNextTime {
		return true
	}

	if w.daily && now.Unix() >= w.rotNextTime {
		return true
	}

	return false
}

// NewFileLogWriter creates a new LogWriter which writes to the given file and
// has rotation enabled if rotate is true and set a memory alignment buffer if
// bufSize is non-zero.
//
// If rotate is true, any time a new log file is opened, the old one is renamed
// with a .### extension to preserve it.  The various Set* methods can be used
// to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is:
//   [%D %T] [%L] (%S) %M
func NewFileLogWriter(fname string, rotate bool, bufSize int) *FileLogWriter {
	w := &FileLogWriter{
		rec:       make(chan LogRecord, LogBufferLength),
		rot:       make(chan bool),
		filename:  fname,
		json:      false,
		format:    "[%D %T] [%L] (%S) %M",
		rotate:    rotate,
		daily:     false,
		hourly:    false,
		maxbackup: 999,
	}

	// init LogCloser
	w.LogCloserInit()

	// open the file for the first time
	if err := w.initOpen(bufSize); err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(filename:%q, bufSize:%d): %s\n", w.filename, bufSize, err)
		return nil
	}

	go func() {
		// 关闭文件
		defer func() {
			if w.file != nil {
				fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
				w.file.Close()
			}
		}()

		for {
			select {
			case <-w.rot: // 外部调用Rotate函数，切割log文件
				if err := w.intRotate(); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
					return
				}

			case rec, ok := <-w.rec:
				if !ok { // rec channel关闭了，退出这个log输出goroutine
					return
				}

				if w.IsClosed(rec) {
					return
				}

				if !w.caller {
					rec.Source = ""
				}

				// 满足相关设定，切割文件了
				if w.checkRotate() {
					if err := w.intRotate(); err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						return
					}
				}

				// Perform the write
				// 写log
				var recStr string
				if !w.json {
					recStr = FormatLogRecord(w.format, &rec)
				} else {
					//recJson, _ := json.Marshal(rec)
					//recStr = String(recJson)
					recBytes := append(rec.JSON(), Slice(newLine)...)
					recStr = String(recBytes)
				}
				n, err := fmt.Fprint(w.file, recStr)
				if err != nil {
					// Recreate log file when the file was lost
					_, err := os.Stat(w.filename)
					if nil != err && os.IsNotExist(err) {
						w.file.Close()
						fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
						if err != nil {
							fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
							return
						}
						w.file = fd
					}

					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
					return
				}

				// Update the counts
				w.curlines++
				w.cursize += int64(n)
			}

		}
	}()

	return w
}

// Request that the logs rotate
func (w *FileLogWriter) Rotate() {
	w.rot <- true
}

// If this is called in a threaded context, it MUST be synchronized
// 关闭旧的文件，按照设置rename新的名称，再打开一个文件
func (w *FileLogWriter) intRotate() error {
	// Close any log file that may be open
	// 切割日志文件前，先把当前打开的日志文件关闭
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}

	// If we are keeping log files, move it to the next available number
	// 为旧文件找到一个合适的名称
	now := time.Now()
	if w.rotate {
		_, err := os.Lstat(w.filename)
		if err == nil { // file exists
			// Find the next available number
			num := 1
			fname := ""
			if w.hourly && now.Unix() >= w.rotNextTime {
				year, month, day := now.Date()
				hour, _, _ := now.Clock()
				preTime := time.Date(year, month, day, hour-w.rotInterval, 0, 0, 0, time.Local)

				suffix := fmt.Sprintf("%s", preTime.Format(w.rotSuffix))
				for ; err == nil && num <= w.maxbackup; num++ {
					fname = w.filename + fmt.Sprintf(".%s.%03d", suffix, num)
					_, err = os.Lstat(fname)
				}
				// return error if the last file checked still existed
				if err == nil {
					return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.filename)
				}
			} else if w.daily && now.Unix() >= w.rotNextTime {
				// daily 复用旧逻辑
				yesterday := now.AddDate(0, 0, -1).Format(w.rotSuffix)

				for ; err == nil && num <= w.maxbackup; num++ {
					fname = w.filename + fmt.Sprintf(".%s.%03d", yesterday, num)
					_, err = os.Lstat(fname)
				}
				// return error if the last file checked still existed
				if err == nil {
					return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.filename)
				}
			} else {
				// maxlines or maxsize
				num = w.maxbackup - 1
				for ; num >= 1; num-- {
					fname = w.filename + fmt.Sprintf(".%d", num)
					nfname := w.filename + fmt.Sprintf(".%d", num+1)
					_, err = os.Lstat(fname)
					if err == nil {
						// 通过文件替换的方式删除序号最大的旧文件
						os.Rename(fname, nfname)
					}
				}
			}

			w.file.Close()

			//err = os.Rename(w.filename, fname)
			//if err != nil {
			//	return fmt.Errorf("Rotate: %s\n", err)
			//}

			// Rename the file to its newfound home
			for i := 0; i < 150; i++ {
				// For loop to handle failure of rename when file is accessed by agent
				// 通过文件替换的方式删除旧文件
				err = os.Rename(w.filename, fname)
				if err != nil {
					fmt.Println("Rename clash - Retrying")
					time.Sleep(100e6) // sleep 100 ms
					continue
				}

				break
			}
		}
	}

	// Open the log file
	// 打开新的文件
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		panic(err) // 打不开文件，说明fd耗完或者磁盘耗尽
		return err
	}
	w.file = fd

	// Set the daily open date to the current date
	if w.hourly {
		year, month, day := now.Date()
		hour, _, _ := now.Clock()
		w.rotNextTime = time.Date(year, month, day, hour+w.rotInterval, 0, 0, 0, time.Local).Unix()
	}
	if w.daily {
		year, month, day := now.Date()
		hour, _, _ := now.Clock()
		w.rotNextTime = time.Date(year, month, day+1, hour, 0, 0, 0, time.Local).Unix()
	}

	w.cursize = 0
	// 防止程序重启创建一个新的日志文件
	if fstat, err := fd.Stat(); nil == err && nil != fstat {
		w.cursize = fstat.Size()
		now = fstat.ModTime()
	}
	// initialize rotation values
	w.curlines = 0

	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	return nil
}

func createDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

// If this is called in a threaded context, it MUST be synchronized
func (w *FileLogWriter) initOpen(bufSize int) error {
	// Already opened
	if w.file != nil {
		return nil
	}

	// 创建文件所在的路径
	path := filepath.Dir(w.filename)
	// filepath.Dir("hello.log") = "."
	// filepath.Dir("./hello.log") = "."
	// filepath.Dir("../hello.log") = "..
	if path != "" && path != "." && path != ".." {
		if err := createDir(path); err != nil {
			return err
		}
	}

	// Open the log file
	if bufSize == 0 {
		fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		w.file = fd
	} else {
		writer := newBufFileWriter(w.filename, bufSize)
		if _, err := writer.open(); err != nil {
			return err
		}
		w.file = writer
	}

	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	// Set the daily/hourly open date to the current date
	year, month, day := now.Date()
	hour, _, _ := now.Clock()
	w.rotNextTime = time.Date(year, month, day+1, hour, 0, 0, 0, time.Local).Unix()

	// initialize rotation values
	w.curlines = 0
	w.cursize = 0

	fstat, err := os.Lstat(w.filename)
	if err == nil {
		w.cursize = fstat.Size()
	}

	return nil
}

// Set the logging json format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetJson(jsonFormat bool) *FileLogWriter {
	w.json = jsonFormat
	return w
}

// Set the logging format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}

// Set the logfile header and footer (chainable).  Must be called before the first log
// message is written.  These are formatted similar to the FormatLogRecord (e.g.
// you can use %D and %T in your header/footer for date and time).
func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.curlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

// Set rotate at linecount (chainable). Must be called before the first log
// message is written.
func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	if maxlines > 0 {
		w.maxlines = maxlines
	}
	return w
}

// Set rotate at size (chainable). Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateSize(maxsize int64) *FileLogWriter {
	if maxsize > 0 {
		w.maxsize = maxsize
	}
	return w
}

// Set rotate daily (chainable). Must be called before the first log message is
// written.
func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	w.daily = daily

	if w.daily {
		return w.SetRotateParams(ROT_TYPE_DAY, "", 0)
	}

	return w
}

// Set rotate daily (chainable). Must be called before the first log message is
// written.
func (w *FileLogWriter) SetRotateHourly(rotHours int) *FileLogWriter {
	w.hourly = false

	if 0 < rotHours {
		return w.SetRotateParams(ROT_TYPE_HOUR, "", rotHours)
	}

	return w
}

// Set rotate params
func (w *FileLogWriter) SetRotateParams(rtype RotType, suffix string, interval int) *FileLogWriter {
	if !w.rotate {
		return w
	}

	w.daily = false
	w.hourly = false

	w.rotSuffix = suffix
	w.rotInterval = interval

	now := time.Now()
	year, month, day := now.Date()
	hour, _, _ := now.Clock()

	switch rtype {
	case ROT_TYPE_DAY:
		if len(w.rotSuffix) == 0 {
			w.rotSuffix = DEF_ROT_TIME_SUFFIX_DAY
		}

		w.rotNextTime = time.Date(year, month, day+1, hour, 0, 0, 0, time.Local).Unix()

		// 开关参数设置一定要放在终止时间 dailyOpenDate 之后
		w.daily = true
		w.hourly = false

	case ROT_TYPE_HOUR:
		if len(w.rotSuffix) == 0 {
			w.rotSuffix = DEF_ROT_TIME_SUFFIX_HOUR
		}
		if w.rotInterval <= 0 {
			w.rotInterval = 1
		}

		w.rotNextTime = time.Date(year, month, day, hour+w.rotInterval, 0, 0, 0, time.Local).Unix()

		// 开关参数设置一定要放在终止时间 dailyOpenDate 之后
		// 因为为了兼容以往 log4go 函数 NewFileLogWriter，file log writer 的参数设置函数 SetXX 是在 NewFileLogWriter
		// 之后调用的，此处先设置参数再打开开关，是为了防止极端 case：本函数如果先把 hourly or daily 开关打开，而 rotNextTime
		// or dailyOpenDate 还未设置，则很容易造成一个日志还没写就对日志文件进行了切割
		w.hourly = true
		w.daily = false
	}

	return w
}

// Set max backup files. Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateMaxBackup(maxbackup int) *FileLogWriter {
	if maxbackup > 0 {
		w.maxbackup = maxbackup
	}
	return w
}

// SetRotate changes whether or not the old logs are kept. (chainable) Must be
// called before the first log message is written.  If rotate is false, the
// files are overwritten; otherwise, they are rotated to another file before the
// new log is opened.
func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	w.rotate = rotate
	return w
}

// Set whether output the filename/function name/line number info or not.
// Must be called before the first log message is written.
func (w *FileLogWriter) SetCallerFlag(flag bool) *FileLogWriter {
	w.caller = flag
	if !w.caller {
		// this is a xml log writer
		if strings.Contains(w.format, "<timestamp>%D %T</timestamp>") {
			w.SetFormat(
				`  <record level="%L">
    <timestamp>%D %T</timestamp>
    <message>%M</message>
  </record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
		}
	}
	return w
}

// NewXMLLogWriter is a utility method for creating a FileLogWriter set up to
// output XML record log messages instead of line-based ones.
func NewXMLLogWriter(fname string, rotate bool, bufSize int) *FileLogWriter {
	return NewFileLogWriter(fname, rotate, bufSize).SetFormat(
		`  <record level="%L">
    <timestamp>%D %T</timestamp>
    <source>%S</source>
    <message>%M</message>
  </record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
}
