package emt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
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

////////////////////////////////////////////////////////////////////////
//
// an implementation to annotate errors with timestamps

type timeAnnotatingCatcher struct {
	mu       sync.RWMutex
	errs     []*timestampError
	maxSize  int
	extended bool
}

// NewTimestampCatcher produces a Catcher instance that reports the
// short form of all constituent errors and annotates those errors
// with a timestamp to reflect when the error was collected.
func NewTimestampCatcher() Catcher { return MakeTimestampCatcher(0) }

// NewExtendedTimestampCatcher adds long-form annotation to the
// aggregated error message (e.g. including stacks, when possible.)
func NewExtendedTimestampCatcher() Catcher { return MakeExtendedTimestampCatcher(0) }

// MakeTimestampCatcher constructs a Catcher instance that annotates
// all errors with their collection time; however, if the size is
// greater than 0 the catcher will never collect more than the
// specified number of errors, discarding earlier messages when adding
// new messages.
func MakeTimestampCatcher(size int) Catcher {
	if size < 0 {
		size = 0
	}

	return &timeAnnotatingCatcher{
		errs:    make([]*timestampError, 0, size),
		maxSize: size,
	}
}

// MakeTimestampCatcher constructs a Catcher instance that annotates
// all errors with their collection time and also captures stacks when
// possible. If the size greater than 0 the catcher will never collect
// more than the specified number of errors, discarding earlier
// messages when adding new messages.
func MakeExtendedTimestampCatcher(size int) Catcher {
	if size < 0 {
		size = 0
	}
	return &timeAnnotatingCatcher{
		errs:     make([]*timestampError, 0, size),
		maxSize:  size,
		extended: true,
	}
}

func (c *timeAnnotatingCatcher) Add(err error) {
	if err == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.safeAdd(err)
}

func (c *timeAnnotatingCatcher) safeAdd(err error) {
	switch e := err.(type) {
	case nil:
	case *timestampError:
		if c.maxSize <= 0 || c.maxSize > len(c.errs) {
			c.errs = append(c.errs, e)
		} else {
			c.errs = c.errs[1:]
			c.errs = append(c.errs, e)
		}
	case error:
		c.safeAdd(newTimeStampError(e).setExtended(c.extended))
	}
}

func (c *timeAnnotatingCatcher) Extend(errs []error) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, err := range errs {
		if err == nil {
			continue
		}

		c.safeAdd(newTimeStampError(err).setExtended(c.extended))
	}
}

func (c *timeAnnotatingCatcher) AddWhen(cond bool, err error) {
	if !cond {
		return
	}

	c.Add(err)
}

func (c *timeAnnotatingCatcher) ExtendWhen(cond bool, errs []error) {
	if !cond {
		return
	}

	c.Extend(errs)
}

func (c *timeAnnotatingCatcher) New(e string) {
	if e == "" {
		return
	}

	c.Add(errors.New(e))
}

func (c *timeAnnotatingCatcher) NewWhen(cond bool, e string) {
	if !cond {
		return
	}

	c.New(e)
}

func (c *timeAnnotatingCatcher) Errorf(f string, args ...interface{}) {
	if f == "" {
		return
	} else if len(args) == 0 {
		c.New(f)
		return
	}

	c.Add(fmt.Errorf(f, args...))
}

func (c *timeAnnotatingCatcher) ErrorfWhen(cond bool, f string, args ...interface{}) {
	if !cond {
		return
	}

	c.Errorf(f, args...)
}

func (c *timeAnnotatingCatcher) Check(fn CheckFunction) {
	c.Add(fn())
}

func (c *timeAnnotatingCatcher) CheckWhen(cond bool, fn CheckFunction) {
	if !cond {
		return
	}

	c.Add(fn())
}

func (c *timeAnnotatingCatcher) CheckExtend(fns []CheckFunction) {
	for _, fn := range fns {
		c.Add(fn())
	}
}

func (c *timeAnnotatingCatcher) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errs)
}

func (c *timeAnnotatingCatcher) Cap() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return cap(c.errs)
}

func (c *timeAnnotatingCatcher) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errs) > 0
}

func (c *timeAnnotatingCatcher) Errors() []error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]error, len(c.errs))
	for idx, err := range c.errs {
		out[idx] = err
	}

	return out
}

func (c *timeAnnotatingCatcher) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	output := make([]string, len(c.errs))

	for idx, err := range c.errs {
		if err.extended {
			output[idx] = err.String()
		} else {
			output[idx] = err.String()
		}
	}

	return strings.Join(output, "\n")
}

func (c *timeAnnotatingCatcher) Resolve() error {
	if !c.HasErrors() {
		return nil
	}

	return errors.New(c.String())
}
