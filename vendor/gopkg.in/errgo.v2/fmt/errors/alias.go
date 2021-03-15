package errors

import (
	"gopkg.in/errgo.v2/errors"
)

type (
	Causer  = errors.Causer
	Wrapper = errors.Wrapper
	Locator = errors.Locator
)

// Deprecated: Locationer is the old name for Locator,
// kept for backward compatibility only.
type Locationer = errors.Locationer

// New returns a new error with the given error message and no cause. It
// is a drop-in replacement for errors.New from the standard library.
func New(s string) error {
	err := errors.New(s)
	errors.SetLocation(err, 1)
	return err
}

// Note returns a new error that wraps the given error
// and holds information about the caller. It preserves the cause if
// shouldPreserveCause is non-nil and shouldPreserveCause(err) is true. If msg is
// non-empty, it is used to prefix the returned error's string value.
//
// If err is nil, Note returns nil without calling shouldPreserveCause.
func Note(err error, shouldPreserveCause func(error) bool, msg string) error {
	err = errors.Note(err, shouldPreserveCause, msg)
	errors.SetLocation(err, 1)
	return err
}

// Because returns a new error that wraps err and has the given cause,
// and adds the given message.
//
// If err is nil and msg is empty, the returned error's message will be
// the same as cause's. This is equivalent to calling errors.Note(cause,
// errors.Is(cause), "") and is useful for annotating an global error
// value with source location information.
//
// Because returns a nil error if all of err, cause and msg are
// zero.
func Because(err, cause error, msg string) error {
	err = errors.Because(err, cause, msg)
	errors.SetLocation(err, 1)
	return err
}

// Wrap returns a new error that wraps the given error and holds
// information about the caller. It is equivalent to Note(err, nil, "").
// Note that this means that the returned error has no cause.
func Wrap(err error) error {
	err = errors.Wrap(err)
	errors.SetLocation(err, 1)
	return err
}

// Cause returns the cause of the given error.  If err does not
// implement Causer or its Cause method returns nil, it returns err itself.
//
// Cause is the usual way to diagnose errors that may have
// been wrapped.
func Cause(err error) error {
	return errors.Cause(err)
}

// SetLocation sets the location of the error to the file and line number
// of the code that is running callDepth frames above
// the caller. It does nothing if the error was not created
// by this package. SetLocation should only be called
// mmediately after an error has been created, before
// it is shared with other code which could access it
// concurrently.
func SetLocation(err error, callDepth int) {
	errors.SetLocation(err, callDepth+1)
}

// Details returns information about the stack of
// underlying errors wrapped by err, in the format:
//
// 	[{filename:99: error one} {otherfile:55: cause of error one}]
//
// The details are found by type-asserting the error to
// the Locator, Causer and Wrapper interfaces.
// Details of the underlying stack are found by
// recursively calling Underlying when the
// underlying error implements Wrapper.
func Details(err error) string {
	return errors.Details(err)
}

// Is returns a function that returns whether the
// an error is equal to the given error.
// It is intended to be used as a "shouldPreserveCause" argument
// to Note. For example:
//
// 	return errgo.Note(err, errgo.Is(http.ErrNoCookie), "")
//
// would return an error with an http.ErrNoCookie cause
// only if that was err's cause.
func Is(err error) func(error) bool {
	return errors.Is(err)
}

// Any returns true. It can be used as an argument to Note
// to allow any cause to pass through to the wrapped
// error.
func Any(err error) bool {
	return errors.Any(err)
}
