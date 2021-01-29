package errors

import "runtime"

// errorInfo holds a description of an error along with information about
// where the error was created.
//
// It may be embedded  in custom error types to add
// extra information that this errors package can
// understand.
type errorInfo struct {
	// message holds the text of the error message. It may be empty
	// if underlying is set.
	message string

	// cause holds the cause of the error as returned
	// by the Cause method.
	cause error

	// underlying holds the underlying error, if any.
	underlying error

	// callerPC holds the program counter of the calling
	// code that created the error. For some reason, a single
	// caller PC is not enough to allow CallersFrames
	// to extract a single source location.
	callerPC [2]uintptr
}

func newError(underlying, cause error, msg string) error {
	err := &errorInfo{
		underlying: underlying,
		cause:      cause,
		message:    msg,
	}
	// Skip two frames as we're interested in the caller of the
	// caller of newError.
	err.setLocation(2)
	return err
}

// Underlying returns the underlying error if any.
func (e *errorInfo) Underlying() error {
	return e.underlying
}

// Cause implements Causer.
func (e *errorInfo) Cause() error {
	return e.cause
}

// Message returns the top level error message.
func (e *errorInfo) Message() string {
	return e.message
}

// Error implements error.Error.
func (e *errorInfo) Error() string {
	switch {
	case e.message == "" && e.underlying == nil:
		return "<no error>"
	case e.message == "":
		return e.underlying.Error()
	case e.underlying == nil:
		return e.message
	}
	return e.message + ": " + e.underlying.Error()
}

// GoString returns the details of the receiving error message, so that
// printing an error with %#v will produce useful information.
func (e *errorInfo) GoString() string {
	return Details(e)
}

func (e *errorInfo) setLocation(callDepth int) {
	// Technically this might not be correct in the future
	// because with mid-stack inlining, the skip count
	// might not directly correspond to the number of
	// frames to skip.
	//
	// If this fails, we'll leave the location as is,
	// which seems reasonable.
	var callerPC [2]uintptr
	if runtime.Callers(callDepth+2, callerPC[:]) == len(callerPC) {
		e.callerPC = callerPC
	}
}

// Location implements Locator.
func (e *errorInfo) Location() (file string, line int) {
	frames := runtime.CallersFrames(e.callerPC[:])
	frame, ok := frames.Next()
	if ok {
		return frame.File, frame.Line
	}
	return "", 0
}
