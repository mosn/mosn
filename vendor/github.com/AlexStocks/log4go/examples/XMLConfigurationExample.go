package main

import (
	"os"
	"time"
)

import (
	l4g "github.com/AlexStocks/log4go"
)

func main() {
	// Load the configuration (isn't this easy?)
	l4g.LoadConfiguration("example.xml")
	defer l4g.Close() // 20160921添加此行代码，否则日志无法输出
	time.Sleep(2e9)   // wait send out udp package

	// And now we're ready!
	l4g.Finest("This will only go to those of you really cool UDP kids!  If you change enabled=true.")
	l4g.Debug("Oh no!  %d + %d = %d!", 2, 2, 2+2)
	l4g.Info("About that time, eh chaps?")

	os.Remove("trace.xml")
	os.Remove("test.log")
}
