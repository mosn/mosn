package log

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-syslog"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var (
	DefaultLogger, StartLogger *logger

	remoteSyslogPrefixes = map[string]string{
		"syslog+tcp://": "tcp",
		"syslog+udp://": "udp",
		"syslog://":     "udp",
	}

	//due to the fact that multiple access log owned by different listener can point to same log path
	//make logger instance for each log path unique, and can be shared by different access log
	// AccessLog x(path:/home/a, format:foo) -> RealLog a
	// AccessLog y(path:/home/a, format:bar) -> RealLog a
	// AccessLog z(path:/home/b)             -> RealLog b
	loggers []*logger
)

func init() {
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
	*log.Logger

	Output  string
	Level   LogLevel
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

	loggers = append(loggers, DefaultLogger)

	return DefaultLogger.Start()
}

func ByContext(ctx context.Context) Logger {
	if ctx != nil {
		if logger := ctx.Value(types.ContextKeyLogger); logger != nil {
			return logger.(Logger)
		}
	}
	
	if DefaultLogger == nil {
		InitDefaultLogger("",DEBUG)
	}
	
	return DefaultLogger
}

func GetLoggerInstance(output string, level LogLevel) (Logger, error) {
	for _, logger := range loggers {
		if logger.Output == output && logger.Level == level {
			return logger, nil
		}
	}

	return NewLogger(output, level)
}

func NewLogger(output string, level LogLevel) (Logger, error) {
	logger := &logger{
		Output:  output,
		Level:   level,
		Roller:  DefaultLogRoller(),
		fileMux: new(sync.RWMutex),
	}

	loggers = append(loggers, logger)

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
		l.Printf(InfoPre+format, args...)
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.Level >= DEBUG {
		l.Printf(DebugPre+format, args...)
	}
}

func (l *logger) Warnf(format string, args ...interface{}) {
	if l.Level >= WARN {
		l.Printf(WarnPre+format, args...)
	}
}

func (l *logger) Errorf(format string, args ...interface{}) {
	if l.Level >= ERROR {
		l.Printf(ErrorPre+format, args...)
	}
}

func (l *logger) Tracef(format string, args ...interface{}) {
	if l.Level >= TRACE {
		l.Printf(TracePre+format, args...)
	}
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	if l.Level >= FATAL {
		l.Printf(FatalPre+format, args...)
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

func Reopen() error {
	for _, logger := range loggers {
		if err := logger.Reopen(); err != nil {
			return err
		}
	}

	return nil
}

func CloseAll() error {
	for _, logger := range loggers {
		if err := logger.Close(); err != nil {
			return err
		}
	}

	return nil
}
