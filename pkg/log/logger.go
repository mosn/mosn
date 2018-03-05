package log

import (
	"io"
	"log"
	"sync"
	"os"
	"strings"
	"github.com/hashicorp/go-syslog"
)

var remoteSyslogPrefixes = map[string]string{
	"syslog+tcp://": "tcp",
	"syslog+udp://": "udp",
	"syslog://":     "udp",
}

var DefaultLogger *logger

// Logger
type logger struct {
	Output  string
	Level   LogLevel
	*log.Logger
	Roller  *LogRoller
	writer  io.Writer
	fileMux *sync.RWMutex
}

func InitDefaultLogger(output string, level LogLevel) error {
	DefaultLogger = &logger{
		Output: output,
		Level:  level,
		Roller: DefaultLogRoller(),
	}

	return DefaultLogger.Start()
}

func NewLogger(output string, level LogLevel) (Logger, error) {
	logger := &logger{
		Output: output,
		Level:  level,
		Roller: DefaultLogRoller(),
	}

	return logger, logger.Start()
}

func (l *logger) Start() error {
	// initialize mutex on start
	l.fileMux = new(sync.RWMutex)

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

	l.Logger = log.New(l.writer, "", 0)

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
