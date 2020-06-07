package getty

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger for user who want to customize logger of getty
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})

	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
}

type LoggerLevel int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	LoggerLevelDebug = LoggerLevel(zapcore.DebugLevel)
	// InfoLevel is the default logging priority.
	LoggerLevelInfo = LoggerLevel(zapcore.InfoLevel)
	// WarnLevel logs are more important than Infof, but don't need individual
	// human review.
	LoggerLevelWarn = LoggerLevel(zapcore.WarnLevel)
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	LoggerLevelError = LoggerLevel(zapcore.ErrorLevel)
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	LoggerLevelDPanic = LoggerLevel(zapcore.DPanicLevel)
	// PanicLevel logs a message, then panics.
	LoggerLevelPanic = LoggerLevel(zapcore.PanicLevel)
	// FatalLevel logs a message, then calls os.Exit(1).
	LoggerLevelFatal = LoggerLevel(zapcore.FatalLevel)
)

var (
	log       Logger
	zapLogger *zap.Logger

	zapLoggerConfig        = zap.NewDevelopmentConfig()
	zapLoggerEncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
)

func init() {
	zapLoggerConfig.EncoderConfig = zapLoggerEncoderConfig
	zapLogger, _ = zapLoggerConfig.Build()
	log = zapLogger.Sugar()

	// todo: flushes buffer when redirect log to file.
	// var exitSignal = make(chan os.Signal)
	// signal.Notify(exitSignal, syscall.SIGTERM, syscall.SIGINT)
	// go func() {
	// 	<-exitSignal
	// 	// Sync calls the underlying Core's Sync method, flushing any buffered log
	// 	// entries. Applications should take care to call Sync before exiting.
	// 	err := zapLogger.Sync() // flushes buffer, if any
	// 	if err != nil {
	// 		fmt.Printf("zapLogger sync err: %+v", perrors.WithStack(err))
	// 	}
	// 	os.Exit(0)
	// }()
}

// SetLogger: customize yourself logger.
func SetLogger(logger Logger) {
	log = logger
}

// GetLogger get getty logger
func GetLogger() Logger {
	return log
}

// SetLoggerLevel
func SetLoggerLevel(level LoggerLevel) error {
	var err error
	zapLoggerConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(level))
	zapLogger, err = zapLoggerConfig.Build()
	if err != nil {
		return err
	}
	log = zapLogger.Sugar()
	return nil
}

// SetLoggerCallerDisable: disable caller info in production env for performance improve.
// It is highly recommended that you execute this method in a production environment.
func SetLoggerCallerDisable() error {
	var err error
	zapLoggerConfig.Development = false
	zapLoggerConfig.DisableCaller = true
	zapLogger, err = zapLoggerConfig.Build()
	if err != nil {
		return err
	}
	log = zapLogger.Sugar()
	return nil
}
