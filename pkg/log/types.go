package log

type LogLevel uint8

const (
	FATAL LogLevel = iota
	ERROR
	WARN
	INFO
	DEBUG
)

type Logger interface {
	Println(args ...interface{})

	Printf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Debugf(format string, args ...interface{})

	Warnf(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Close() error

	Reopen() error
}
