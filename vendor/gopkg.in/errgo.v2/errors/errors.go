// Copyright 2014 Roger Peppe.
// See LICENCE file for details.

// The errors package provides a way to create and diagnose errors. It
// is compatible with the usual Go error idioms but adds a way to wrap
// errors so that they record source location information while
// retaining a consistent way for code to inspect errors to find out
// particular problems.
//
// An error created by this package holds three values, any one of
// which (but not all) may be zero
//
//	- an error message. This is used as part of the result of the Error method.
//	- an underlying error. This holds the error that the error was created in response to.
//	- an error cause. See below.
//
// Error Causes
//
// The "cause" of an error is something that code can use to diagnose
// what an error means and take action accordingly.
//
// For example, if you use os.Remove to remove a file that is not there,
// it returns an error with a cause that satisfies os.IsNotExist. If you
// use filepath.Match to try to match a malformed pattern, it returns an
// error with a cause that equals filepath.ErrBadPattern.
//
// When the errors package wraps an error value with another one, the
// new error hides the cause by default. This is to prevent unintended
// dependencies between unrelated parts of the code base. When the cause
// is hidden, a caller can't write code that relies on the type or value
// of a particular cause which may well only be an implementation detail
// and subject to future change - a potentially API-breaking change if
// the cause is not hidden.
//
// The Note function can be used to preserve an existing error cause.
// The Because function can be used to associate an error with an
// existing cause value.
//
// Error wrapping
//
// When an error is returned that "wraps" another one, the new error
// records the source code location of the caller. This means that it is
// possible to extract this information later (by calling Details, for
// example). It is good practice to wrap errors whenever returning an
// error returned by a lower level function, so that the path taken by
// the error is recorded. This can make debugging significantly easier
// when there are many places an error could have come from.
package errors

// New returns a new error with the given error message and no cause. It
// is a drop-in replacement for errors.New from the standard library.
func New(s string) error {
	return newError(nil, nil, s)
}

// Note returns a new error that wraps the given error. It preserves the cause if
// shouldPreserveCause is non-nil and shouldPreserveCause(err) is true. If msg is
// non-empty, it is used to prefix the returned error's Error value.
//
// If err is nil, Note returns nil and does not call shouldPreserveCause.
func Note(err error, shouldPreserveCause func(error) bool, msg string) error {
	if err == nil {
		return nil
	}
	if shouldPreserveCause == nil {
		return newError(err, nil, msg)
	}
	if cause := Cause(err); shouldPreserveCause(cause) {
		return newError(err, cause, msg)
	}
	return newError(err, nil, msg)
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
	if err == nil && msg == "" {
		if cause == nil {
			return nil
		}
		msg = cause.Error()
	}
	return newError(err, cause, msg)
}

// Wrap returns a new error that wraps the given error and holds
// information about the caller. It is equivalent to Note(err, nil, "").
// Note that this means that the returned error has no cause.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return newError(err, nil, "")
}

// Cause returns the cause of the given error.  If err does not
// implement Causer or its Cause method returns nil, it returns err itself.
//
// Cause is the usual way to diagnose errors that may have
// been wrapped.
func Cause(err error) error {
	if err, ok := err.(Causer); ok {
		if cause := err.Cause(); cause != nil {
			return cause
		}
	}
	return err
}

// SetLocation sets the location of the error to the file and line
// number of the code that is running callDepth frames above the caller.
// It does nothing if the error was not created by this package.
// SetLocation should only be called immediately after an error has been
// created, before it is shared with other code which could access it
// concurrently.
//
// This function is helpful when a helper function is used to
// create an error and the caller of the helper function is
// the important thing, not the location in the helper function itself.
func SetLocation(err error, callDepth int) {
	if err, ok := err.(*errorInfo); ok {
		err.setLocation(callDepth + 1)
	}
}

// Details returns information about the stack of underlying errors
// wrapped by err, in the format:
//
// 		[
//			{filename:99: error one}
//			{otherfile:55: cause of error one}
//		]
//
// The details are found by type-asserting the error to the Locator,
// Causer and Wrapper interfaces. Details of the underlying stack are
// found by recursively calling Underlying when the underlying error
// implements Wrapper.
func Details(err error) string {
	if err == nil {
		return "[]"
	}
	s := make([]byte, 0, 30)
	s = append(s, '[')
	for {
		s = append(s, "\n\t{"...)
		if err, ok := err.(Locator); ok {
			file, line := err.Location()
			if file != "" {
				s = append(s, file...)
				s = append(s, ':')
				s = appendInt(s, line)
				s = append(s, ": "...)
			}
		}
		if cerr, ok := err.(Wrapper); ok {
			s = append(s, cerr.Message()...)
			err = cerr.Underlying()
		} else {
			s = append(s, err.Error()...)
			err = nil
		}
		s = append(s, '}')
		if err == nil {
			break
		}
	}
	s = append(s, "\n]"...)
	return string(s)
}

// Is returns a function that returns whether the an error is equal to
// the given error. It is intended to be used as a "shouldPreserveCause"
// argument to Note. For example:
//
// 	return errgo.Note(err, errgo.Is(http.ErrNoCookie), "")
//
// would return an error with an http.ErrNoCookie cause
// only if that was err's cause.
func Is(err error) func(error) bool {
	return func(err1 error) bool {
		return err == err1
	}
}

// Any returns true. It can be used as an argument to Note
// to allow any cause to pass through to the wrapped
// error.
func Any(error) bool {
	return true
}

// OneOf returns a function suitable for passing to Note
// that reports whether an error matches any of the
// given predicates.
func OneOf(predicates ...func(error) bool) func(error) bool {
	return func(err error) bool {
		for _, predicate := range predicates {
			if predicate(err) {
				return true
			}
		}
		return false
	}
}
