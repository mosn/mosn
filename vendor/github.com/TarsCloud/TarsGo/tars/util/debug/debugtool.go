package debugutil

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

//write2File write byte array to file
//logname: prefix of the file's name, which is like logname.20060102-150405, and the file
//will locates in the same diretory as the bin file which calls it
func write2File(buf []byte, logname string) error {
	path, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.Chdir(path)
	logpath := fmt.Sprintf("%s.%s", logname, time.Now().Format("20060102-150405"))

	var err error
	var file *os.File
	if file, err = os.OpenFile(logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644); err == nil {
		defer file.Close()
		routinenum := fmt.Sprintf("current running goroutine num %d!\n\n", runtime.NumGoroutine())
		if _, err := file.WriteString(routinenum); err == nil {
			writeObj := bufio.NewWriterSize(file, 4096)
			if _, err := writeObj.Write(buf); err == nil {
				if err := writeObj.Flush(); err == nil {
					fmt.Println("successfully flush write into file")
				}
			}
		}
	}
	return err
}

//DumpStack used to dump stack info to file
//all: true means dumping all running goroutine stack, else only dumping the one that calls the func
//logname: prefix of the file's name, which is like logname.20060102-150405
func DumpStack(all bool, logname string) {
	buf := make([]byte, 1024)
	go func() {
		for {
			//the buf is no more than 64M, because Stack dumps no more than 64M
			n := runtime.Stack(buf, all)
			if n < len(buf) {
				buf = buf[:n] //trim unreadable characters
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		write2File(buf, logname)
	}()
}

//SigNotifyStack register os signals to be notified when to dumpstack
//For example, SigNotifyStack(SIGUSR1, true, "stackinfo"), can dump all goroutine stack when received SIGUSR1 signal by "kill -USR1 pid"
//sig: self defined os signals, like SIGUSR1 SIGUSR2 in linux and darwin but not supoorted in windows
//all: true means dumping all running goroutine stack, else only dumping the one that calls the func
//logname: prefix of the file's name, which is like logname.20060102-150405
func SigNotifyStack(sig os.Signal, all bool, logname string) {
	buf := make([]byte, 1024)
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)

	go func() {
		for range c {
			for {
				//the buf is no more than 64M, because Stack dumps no more than 64M
				n := runtime.Stack(buf, all)
				if n < len(buf) {
					buf = buf[:n] //trim unreadable characters
					break
				}
				buf = make([]byte, 2*len(buf))
			}
			write2File(buf, logname)
		}
	}()
}
