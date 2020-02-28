# a log4go fork maintained by AlexStocks, maybe the most powerful on github

Please see http://log4go.googlecode.com/ for more log4go usages. My personal
package (github.com/AlexStocks/goext/log) wrappered log4go functions further
more which provides the most powerful log4go.

Installation:
- Run `go get -u -v github.com/AlexStocks/log4go`

Usage:

- Add the following import:

``` Go
  import l4g "github.com/AlexStocks/log4go"

  func main() {
  	defer l4g.Close() // to close l4g.Global
  }
```

### get logger

#### Global logger

```go
import l4g "github.com/alecthomas/log4go"

l4g.Info("hello world")
defer l4g.Close()
```

#### NewDefaultLogger

```go
log := l4g.NewDefaultLogger(l4g.INFO)
log.Info("hello world")
defer log.Close()
```

#### l4g.Logger

```go
log := make(l4g.Logger)
defer log.Close()
log.AddFilter("stdout", l4g.DEBUG, l4g.NewConsoleLogWriter())
log.Info("hello world")
```

## output log

```go
l4g.Finest()
l4g.Fine()
l4g.Debug()
l4g.Trace()
l4g.Info()
l4g.Warning()
l4g.Error()
l4g.Critical()
```

## UserGuide

### Level

```go
	FINEST
	FINE
	DEBUG
	TRACE
	INFO
	WARNING
	ERROR
	CRITICAL
```

Feature list:

* Output colorful terminal log string by log level
* Output json log
* Add maxbackup choice in examples.xml to delete out of date log file
* Output escape query string safety
* Add filename to every log line
* Create log path if log path does not exist
* Add caller option to let log4go do not output file/function-name/line-number
* Add %P to output process ID
* Rotate log file daily/hourly
* Support json/xml/yml configuration file
