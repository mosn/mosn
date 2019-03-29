package rogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

//DAY for rotate log by day
const (
	DAY DateType = iota
	HOUR
)

//LogWriter is interface for different writer.
type LogWriter interface {
	Write(v []byte)
	NeedPrefix() bool
}

//ConsoleWriter writes the logs to the console.
type ConsoleWriter struct {
}

//RollFileWriter struct for rotate logs by file size.
type RollFileWriter struct {
	logpath  string
	name     string
	num      int
	size     int64
	currSize int64
	currFile *os.File
	openTime int64
}

//DateWriter rotate logs by date.
type DateWriter struct {
	logpath  string
	name     string
	dateType DateType
	num      int
	currDate string
	currFile *os.File
	openTime int64
}

//HourWriter for rotate logs by hour
type HourWriter struct {
}

//DateType is uint8
type DateType uint8

func reOpenFile(path string, currFile **os.File, openTime *int64) {
	*openTime = currUnixTime
	if *currFile != nil {
		(*currFile).Close()
	}
	of, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err == nil {
		*currFile = of
	} else {
		fmt.Println("open log file error", err)
	}
}
func (w *ConsoleWriter) Write(v []byte) {
	os.Stdout.Write(v)
}

//NeedPrefix shows whether needs the prefix for the console writer.
func (w *ConsoleWriter) NeedPrefix() bool {
	return true
}

//Write for writing []byte to the writter.
func (w *RollFileWriter) Write(v []byte) {
	if w.currFile == nil || w.openTime+10 < currUnixTime {
		fullPath := filepath.Join(w.logpath, w.name+".log")
		reOpenFile(fullPath, &w.currFile, &w.openTime)
	}
	if w.currFile == nil {
		return
	}
	n, _ := w.currFile.Write(v)
	w.currSize += int64(n)
	if w.currSize >= w.size {
		w.currSize = 0
		for i := w.num - 1; i >= 1; i-- {
			var n1, n2 string
			if i > 1 {
				n1 = strconv.Itoa(i - 1)
			}
			n2 = strconv.Itoa(i)
			p1 := filepath.Join(w.logpath, w.name+n1+".log")
			p2 := filepath.Join(w.logpath, w.name+n2+".log")
			if _, err := os.Stat(p1); !os.IsNotExist(err) {
				os.Rename(p1, p2)
			}
		}
		fullPath := filepath.Join(w.logpath, w.name+".log")
		reOpenFile(fullPath, &w.currFile, &w.openTime)
	}
}

//NewRollFileWriter returns a RollFileWriter, rotate logs in sizeMB , and num files are keeped.
func NewRollFileWriter(logpath, name string, num, sizeMB int) *RollFileWriter {
	w := &RollFileWriter{
		logpath: logpath,
		name:    name,
		num:     num,
		size:    int64(sizeMB) * 1024 * 1024,
	}
	fullPath := filepath.Join(logpath, name+".log")
	st, _ := os.Stat(fullPath)
	if st != nil {
		w.currSize = st.Size()
	}
	return w
}

//NeedPrefix shows need prefix or not.
func (w *RollFileWriter) NeedPrefix() bool {
	return true
}

//Write method implement for the DateWriter
func (w *DateWriter) Write(v []byte) {
	if w.currFile == nil || w.openTime+10 < currUnixTime {
		fullPath := filepath.Join(w.logpath, w.name+"_"+w.currDate+".log")
		reOpenFile(fullPath, &w.currFile, &w.openTime)
	}
	if w.currFile == nil {
		return
	}

	w.currFile.Write(v)
	currDate := w.getCurrDate()
	if w.currDate != currDate {
		w.currDate = currDate
		w.cleanOldLogs()
		fullPath := filepath.Join(w.logpath, w.name+"_"+w.currDate+".log")
		reOpenFile(fullPath, &w.currFile, &w.openTime)
	}
}

//NeedPrefix shows whether needs prefix info for DateWriter or not.
func (w *DateWriter) NeedPrefix() bool {
	return true
}

//NewDateWriter returns a writer which keeps logs in hours or day format.
func NewDateWriter(logpath, name string, dateType DateType, num int) *DateWriter {
	w := &DateWriter{
		logpath:  logpath,
		name:     name,
		num:      num,
		dateType: dateType,
	}
	w.currDate = w.getCurrDate()
	return w
}

func (w *DateWriter) cleanOldLogs() {
	format := "20060102"
	duration := -time.Hour * 24
	if w.dateType == HOUR {
		format = "2006010215"
		duration = -time.Hour
	}

	t := time.Now()
	t = t.Add(duration * time.Duration(w.num))
	for i := 0; i < 30; i++ {
		t = t.Add(duration)
		k := t.Format(format)
		fullPath := filepath.Join(w.logpath, w.name+"_"+k+".log")
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			os.Remove(fullPath)
		}
	}
	return
}

func (w *DateWriter) getCurrDate() string {
	if w.dateType == HOUR {
		return currDateHour
	}
	return currDateDay // DAY
}
