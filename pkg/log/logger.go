package log

import (
	"github.com/hashicorp/go-syslog"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"context"
	"fmt"
)

var remoteSyslogPrefixes = map[string]string{
	"syslog+tcp://": "tcp",
	"syslog+udp://": "udp",
	"syslog://":     "udp",
}

var DefaultLogger *logger

var StartLogger  *logger

func init(){
	//use console  as start logger
	StartLogger = &logger{
		Output:  "",
		Level:   DEBUG,
		Roller:  DefaultLogRoller(),
		fileMux: new(sync.RWMutex),
	}

	StartLogger.Start()
}

// Logger
type logger struct {
	Output string
	Level  LogLevel
	*log.Logger
	Roller  *LogRoller
	writer  io.Writer
	fileMux *sync.RWMutex
}

func InitDefaultLogger(output string, level LogLevel) error {
	DefaultLogger = &logger{
		Output:  output,
		Level:   level,
		Roller:  DefaultLogRoller(),
		fileMux: new(sync.RWMutex),
	}

	return DefaultLogger.Start()
}

func ByContext(ctx context.Context) Logger {
	if ctx != nil{
		if logger := ctx.Value(types.ContextKeyLogger); logger != nil{
			return logger.(Logger)
		}
	}
	return DefaultLogger
}

func NewLogger(output string, level LogLevel) (Logger, error) {
	logger := &logger{
		Output:  output,
		Level:   level,
		Roller:  DefaultLogRoller(),
		fileMux: new(sync.RWMutex),
	}

	return logger, logger.Start()
}

func (l *logger) Start() error {
	var err error

selectwriter:
	switch l.Output {
	case "", "stderr":
		l.writer = os.Stderr
	case "stdout":
		l.writer = os.Stdout
	case "syslog":
		l.writer, err = gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "mosn")
		if err != nil {
			return err
		}
	default:
		if address := parseSyslogAddress(l.Output); address != nil {
			l.writer, err = gsyslog.DialLogger(address.network, address.address, gsyslog.LOG_ERR, "LOCAL0", "mosn")

			if err != nil {
				return err
			}

			break selectwriter
		}

		var file *os.File

		//create parent dir if not exists
		err := os.MkdirAll(filepath.Dir(l.Output), 0755)

		fmt.Println(err)

		file, err = os.OpenFile(l.Output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		if l.Roller != nil {
			file.Close()
			l.Roller.Filename = l.Output
			l.writer = l.Roller.GetLogWriter()
		} else {
			l.writer = file
		}
	}

	l.Logger = log.New(l.writer, "", log.LstdFlags)

	return nil
}

func (l *logger) Println(args ...interface{}) {
	l.fileMux.RLock()
	l.Logger.Println(args...)
	l.fileMux.RUnlock()
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.fileMux.RLock()
	l.Logger.Printf(format, args...)
	l.fileMux.RUnlock()
}

func (l *logger) Infof(format string, args ...interface{}) {
	if l.Level >= INFO {
		l.Printf(format, args...)
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.Level >= DEBUG {

		l.Printf(format, args...)
	}
}

func (l *logger) Warnf(format string, args ...interface{}) {
	if l.Level >= WARN {
		l.Printf(format, args...)
	}
}

func (l *logger) Errorf(format string, args ...interface{}) {
	if l.Level >= ERROR {
		l.Printf(format, args...)
	}
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	if l.Level >= FATAL {
		l.Printf(format, args...)
	}
}

func (l *logger) Close() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		l.fileMux.Lock()
		err := closer.Close()
		l.fileMux.Unlock()
		return err
	}

	return nil
}

func (l *logger) Reopen() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		l.fileMux.Lock()
		err := closer.Close()

		if err := l.Start(); err != nil {
			return err
		}

		l.fileMux.Unlock()
		return err
	}

	return nil
}

type syslogAddress struct {
	network string
	address string
}

func parseSyslogAddress(location string) *syslogAddress {
	for prefix, network := range remoteSyslogPrefixes {
		if strings.HasPrefix(location, prefix) {
			return &syslogAddress{
				network: network,
				address: strings.TrimPrefix(location, prefix),
			}
		}
	}

	return nil
}
