// Error Catcher
//
// The Catcher type makes it possible to collect from a group of
// operations and then aggregate them as a single error.
package emt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// CheckFunction are functions which take no arguments and return an
// error.
type CheckFunction func() error

// Catcher is an interface for an error collector for use when
// implementing continue-on-error semantics in concurrent
// operations. The different catcher implementations provided by this
// package differ in the formats applied when constructing their
// output (resolved) error object.
//
// Implementations should provide objects that are safe for access
// from multiple threads.
//
// Catchers implement the error interface, and accordingly the
// Add/Extend methods should be able to flatten/merge catchers.
type Catcher interface {
	Add(error)
	AddWhen(bool, error)

	Extend([]error)
	ExtendWhen(bool, []error)

	New(string)
	NewWhen(bool, string)
	Errorf(string, ...interface{})
	ErrorfWhen(bool, string, ...interface{})

	Check(CheckFunction)
	CheckExtend([]CheckFunction)
	CheckWhen(bool, CheckFunction)

	Resolve() error
	HasErrors() bool

	Len() int
	Errors() []error

	// String returns a string that concatenates the values
	// returned by `.Error()` on all of the constituent errors.
	String() string
}

// multiCatcher provides an interface to collect and coalesse error
// messages within a function or other sequence of operations. Used to
// implement a kind of "continue on error"-style operations. The
// methods on MultiCatatcher are thread-safe.
type baseCatcher struct {
	errs    []error
	maxSize int
	mutex   sync.RWMutex
	fmt.Stringer
}

// NewCatcher returns a Catcher instance that you can use to capture
// error messages and aggregate the errors. For consistency with
// earlier versions NewCatcher is the same as NewExtendedCatcher()
//
// DEPRECATED: use one of the other catcher implementations. See the
// documentation for the Catcher interface for most implementations.
func NewCatcher() Catcher { return NewExtendedCatcher() }

// NewBasicCatcher collects error messages and formats them using a
// new-line separated string of the output of error.Error()
func NewBasicCatcher() Catcher { return makeBasicCatcher(&baseCatcher{}) }

// NewSimpleCatcher collects error messages and formats them using a
// new-line separated string of the string format of the error message
// (e.g. %s).
func NewSimpleCatcher() Catcher { return makeSimpleCatcher(&baseCatcher{}) }

// NewExtendedCatcher collects error messages and formats them using a
// new-line separated string of the extended string format of the
// error message (e.g. %+v).
func NewExtendedCatcher() Catcher { return makeExtCatcher(&baseCatcher{}) }

// MakeBasicCatcher collects error messages and formats them using a
// new-line separated string of the output of error.Error(). If the
// size greater than 0 the catcher will never collect more than the
// specified number of errors, discarding earlier messages when adding
// new messages.
func MakeBasicCatcher(size int) Catcher { return makeBasicCatcher(&baseCatcher{maxSize: size}) }

// MakeSimpleCatcher collects error messages and formats them using a
// new-line separated string of the string format of the error message
// (e.g. %s). If the size greater than 0 the catcher will never
// collect more than the specified number of errors, discarding
// earlier messages when adding new messages.
func MakeSimpleCatcher(size int) Catcher { return makeSimpleCatcher(&baseCatcher{maxSize: size}) }

// MakeExtendedCatcher collects error messages and formats them using
// a new-line separated string of the extended string format of the
// error message (e.g. %+v). If the size greater than 0 the catcher
// will never collect more than the specified number of errors,
// discarding earlier messages when adding new messages.
func MakeExtendedCatcher(size int) Catcher { return makeExtCatcher(&baseCatcher{maxSize: size}) }

// Add takes an error object and, if it's non-nil, adds it to the
// internal collection of errors.
func (c *baseCatcher) Add(err error) {
	if err == nil {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.safeAdd(err)
}

func (c *baseCatcher) safeAdd(err error) {
	if c.maxSize <= 0 || c.maxSize > len(c.errs) {
		c.errs = append(c.errs, err)
	} else {
		c.errs = c.errs[1:]
		c.errs = append(c.errs, err)
	}
}

// Len returns the number of errors stored in the collector.
func (c *baseCatcher) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.errs)
}

// Cap returns the capcity of the underlying storage in the collector.
func (c *baseCatcher) Cap() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return cap(c.errs)
}

// HasErrors returns true if the collector has ingested errors, and
// false otherwise.
func (c *baseCatcher) HasErrors() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.errs) > 0
}

// Extend adds all non-nil errors, passed as arguments to the catcher.
func (c *baseCatcher) Extend(errs []error) {
	if len(errs) == 0 {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, err := range errs {
		if err == nil {
			continue
		}

		c.safeAdd(err)
	}
}

func (c *baseCatcher) Errorf(form string, args ...interface{}) {
	if form == "" {
		return
	} else if len(args) == 0 {
		c.New(form)
		return
	}
	c.Add(fmt.Errorf(form, args...))
}

func (c *baseCatcher) New(e string) {
	if e == "" {
		return
	}
	c.Add(errors.New(e))
}

func (c *baseCatcher) AddWhen(cond bool, err error) {
	if !cond {
		return
	}

	c.Add(err)
}

func (c *baseCatcher) ExtendWhen(cond bool, errs []error) {
	if !cond {
		return
	}

	c.Extend(errs)
}

func (c *baseCatcher) ErrorfWhen(cond bool, form string, args ...interface{}) {
	if !cond {
		return
	}

	c.Errorf(form, args...)
}

func (c *baseCatcher) NewWhen(cond bool, e string) {
	if !cond {
		return
	}

	c.New(e)
}

func (c *baseCatcher) Check(fn CheckFunction) { c.Add(fn()) }

func (c *baseCatcher) CheckWhen(cond bool, fn CheckFunction) {
	if !cond {
		return
	}

	c.Add(fn())
}

func (c *baseCatcher) CheckExtend(fns []CheckFunction) {
	for _, fn := range fns {
		c.Add(fn())
	}
}

func (c *baseCatcher) Errors() []error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	out := make([]error, len(c.errs))

	copy(out, c.errs)

	return out
}

// Resolve returns a final error object for the Catcher. If there are
// no errors, it returns nil, and returns an error object with the
// string form of all error objects in the collector.
func (c *baseCatcher) Resolve() error {
	if !c.HasErrors() {
		return nil
	}

	return errors.New(c.String())
}

////////////////////////////////////////////////////////////////////////
//
// separate implementations of grip.Catcher with different string formatting options.

type basicCatcherConstructor func(*baseCatcher)

func makeExtCatcher(bc *baseCatcher) Catcher    { c := &extendedCatcher{bc}; bc.Stringer = c; return bc }
func makeSimpleCatcher(bc *baseCatcher) Catcher { c := &simpleCatcher{bc}; bc.Stringer = c; return bc }
func makeBasicCatcher(bc *baseCatcher) Catcher  { c := &basicCatcher{bc}; bc.Stringer = c; return bc }

type extendedCatcher struct{ *baseCatcher }

func (c *extendedCatcher) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	output := make([]string, len(c.errs))

	for idx, err := range c.errs {
		output[idx] = fmt.Sprintf("%+v", err)
	}

	return strings.Join(output, "\n")
}

type simpleCatcher struct{ *baseCatcher }

func (c *simpleCatcher) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	output := make([]string, len(c.errs))

	for idx, err := range c.errs {
		output[idx] = fmt.Sprintf("%s", err)
	}

	return strings.Join(output, "\n")
}

type basicCatcher struct{ *baseCatcher }

func (c *basicCatcher) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	output := make([]string, len(c.errs))

	for idx, err := range c.errs {
		output[idx] = err.Error()
	}

	return strings.Join(output, "\n")
}
