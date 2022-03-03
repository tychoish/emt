package emt

import (
	"fmt"
	"time"
)

// ErrorTimeFinder unwraps a timestamp annotated error if possible and
// is capable of finding a timestamp in an error that has been
// annotated using pkg/errors.
func ErrorTimeFinder(err error) (time.Time, bool) {
	if err == nil {
		return time.Time{}, false
	}

	if tserr, ok := err.(*timestampError); ok {
		if tserr == nil {
			return time.Time{}, false
		}
		return tserr.time, true
	}

	for {
		switch e := err.(type) {
		case *timestampError:
			if e == nil {
				return time.Time{}, false
			}
			return e.time, true
		case interface{ Cause() error }:
			err = e.Cause()
			continue
		case interface{ Unwrap() error }:
			err = e.Unwrap()
			continue
		}
		break
	}

	return time.Time{}, false
}

type timestampError struct {
	err      error
	time     time.Time
	extended bool
}

func newTimeStampError(err error) *timestampError {
	if err == nil {
		return nil
	}

	switch v := err.(type) {
	case *timestampError:
		return v
	default:
		return &timestampError{
			err:  err,
			time: time.Now(),
		}
	}
}

func (e *timestampError) setExtended(v bool) *timestampError { e.extended = v; return e }

// WrapErrorTime annotates an error with the timestamp. The underlying
// concrete object implements message.Composer as well as error.
func WrapErrorTime(err error) error { return newTimeStampError(err) }

// WrapErrorTimeMessage annotates an error with the timestamp and a
// string form. The underlying concrete object implements
// message.Composer as well as error.
func WrapErrorTimeMessage(err error, m string) error {
	if err == nil {
		return nil
	}
	return newTimeStampError(fmt.Errorf("%s: %w", m, err))
}

// WrapErrorTimeMessagef annotates an error with a timestamp and a
// string formated message, like fmt.Sprintf or fmt.Errorf. The
// underlying concrete object implements  message.Composer as well as
// error.
func WrapErrorTimeMessagef(err error, m string, args ...interface{}) error {
	return WrapErrorTimeMessage(err, fmt.Sprintf(m, args...))
}

func (e *timestampError) String() string {
	if e.err == nil {
		return ""
	}

	if e.extended {
		return fmt.Sprintf("%+v", e.err)
	}

	return e.err.Error()
}

func (e *timestampError) Cause() error  { return e.err }
func (e *timestampError) Unwrap() error { return e.err }
func (e *timestampError) Error() string {
	return fmt.Sprintf("[%s], %s", e.time.Format(time.RFC3339), e.String())
}

func (e *timestampError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "[%s] %+v", e.time.Format(time.RFC3339), e.Cause())
		}
		fallthrough
	case 's':
		_, _ = fmt.Fprintf(s, "[%s] %s", e.time.Format(time.RFC3339), e.String())
	case 'q':
		_, _ = fmt.Fprintf(s, "[%s] %q", e.time.Format(time.RFC3339), e.String())
	}
}
