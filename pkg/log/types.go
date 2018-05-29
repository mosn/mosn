package log

type LogLevel uint8

const (
	FATAL LogLevel = iota
	ERROR
	WARN
	INFO
	DEBUG
)

const (
	InfoPre  string = "[INFO]"
	DebugPre string = "[DEBUG]"
	WarnPre  string = "[WARN]"
	ErrorPre string = "[ERROR]"
	FatalPre string = "[Fatal]"
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
