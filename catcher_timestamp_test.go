package emt

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func makeAssertHasTimestamp(t *testing.T) func(ts time.Time, ok bool) {
	return func(ts time.Time, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("timstamp should be ok")
		}
		if ts.IsZero() {
			t.Fatal("timestamp should not be zero")
		}
	}
}

func makeAssertNoTimestamp(t *testing.T) func(ts time.Time, ok bool) {
	return func(ts time.Time, ok bool) {
		t.Helper()
		if ok {
			t.Fatal("timstamp should not exist")
		}
		if !ts.IsZero() {
			t.Fatal("empty timestamp should be zero")
		}
	}
}

type causeImpl struct {
	val   string
	cause error
}

func (ci *causeImpl) Cause() error  { return ci.cause }
func (ci *causeImpl) Error() string { return fmt.Sprintf("%s: %v", ci.val, ci.cause) }

func TestTimestampError(t *testing.T) {
	t.Run("ErrorFinder", func(t *testing.T) {
		assertHasTimestamp := makeAssertHasTimestamp(t)
		assertNoTimestamp := makeAssertNoTimestamp(t)

		assertNoTimestamp(
			ErrorTimeFinder(nil),
		)

		assertNoTimestamp(
			ErrorTimeFinder(newTimeStampError(nil)),
		)

		assertHasTimestamp(
			ErrorTimeFinder(newTimeStampError(errors.New("hello"))),
		)

		assertNoTimestamp(
			ErrorTimeFinder(fmt.Errorf("outer wrap: %w", (fmt.Errorf("wrap: %w", errors.New("hello"))))),
		)

		assertHasTimestamp(
			ErrorTimeFinder(fmt.Errorf("wrap: %w", newTimeStampError(errors.New("hello")))),
		)

		assertNoTimestamp(
			ErrorTimeFinder(fmt.Errorf("wrap: %w", newTimeStampError(nil))),
		)

		assertHasTimestamp(
			ErrorTimeFinder(&causeImpl{val: "wrapouter", cause: newTimeStampError(errors.New("hello"))}),
		)
		assertNoTimestamp(
			ErrorTimeFinder(&causeImpl{val: "wrapouter"}),
		)

		assertHasTimestamp(
			ErrorTimeFinder(newTimeStampError(fmt.Errorf("wrap: %w", (errors.New("hello"))))),
		)

		assertHasTimestamp(
			ErrorTimeFinder(fmt.Errorf("wrap: %w", (newTimeStampError(errors.New("hello"))))),
		)
	})
	t.Run("Wrap", func(t *testing.T) {
		t.Run("WithTimestamp", func(t *testing.T) {
			assertHasTimestamp := makeAssertHasTimestamp(t)
			assertNoTimestamp := makeAssertNoTimestamp(t)

			assertNoTimestamp(
				ErrorTimeFinder(nil),
			)
			assertNoTimestamp(
				ErrorTimeFinder(WrapErrorTime(nil)),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTime(errors.New("hello"))),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", (WrapErrorTime(errors.New("hello"))))),
			)
			assertNoTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", (WrapErrorTime(nil)))),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("plainwrap: %w", fmt.Errorf("wrap: %w", (WrapErrorTime(errors.New("hello")))))),
			)
		})
		t.Run("WithTimestampMessage", func(t *testing.T) {
			assertHasTimestamp := makeAssertHasTimestamp(t)
			assertNoTimestamp := makeAssertNoTimestamp(t)

			assertNoTimestamp(
				ErrorTimeFinder(nil),
			)
			assertNoTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessage(nil, "earth")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessage(errors.New("hello"), "earth")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", (WrapErrorTimeMessage(errors.New("hello"), "earth")))),
			)
			assertNoTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", (WrapErrorTimeMessage(nil, "earth")))),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessage(fmt.Errorf("wrap: %w", errors.New("hello")), "msg")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("plainwrap: %w", fmt.Errorf("wrap: %w", WrapErrorTimeMessage(errors.New("hello"), "earth")))),
			)
		})
		t.Run("WithTimestampMessageFormatEmpty", func(t *testing.T) {
			assertHasTimestamp := makeAssertHasTimestamp(t)
			assertNoTimestamp := makeAssertNoTimestamp(t)

			assertNoTimestamp(
				ErrorTimeFinder(nil),
			)
			assertNoTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(nil, "earth")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(errors.New("hello"), "earth")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(errors.New("hello"), "earth")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", WrapErrorTimeMessagef(errors.New("hello"), "earth"))),
			)
			assertNoTimestamp(
				ErrorTimeFinder(fmt.Errorf("wrap: %w", WrapErrorTimeMessagef(nil, "earth"))),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(fmt.Errorf("wrap: %w", errors.New("hello")), "msg")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("plainwrap: %w", fmt.Errorf("wrap: %w", WrapErrorTimeMessagef(errors.New("hello"), "earth")))),
			)
		})
		t.Run("WrapWithTimestampMessageFormat", func(t *testing.T) {
			assertHasTimestamp := makeAssertHasTimestamp(t)
			assertNoTimestamp := makeAssertNoTimestamp(t)

			assertNoTimestamp(
				ErrorTimeFinder(nil),
			)
			assertNoTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(nil, "earth-%s", "lings")),
			)
			assertNoTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(nil, "earth-%q", "lings")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(errors.New("hello"), "earth-%s", "lings")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(errors.New("hello"), "earth-%q", "lings")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("foo: %w", WrapErrorTimeMessagef(errors.New("hello"), "earth-%s", "lings"))),
			)
			assertNoTimestamp(
				ErrorTimeFinder(fmt.Errorf("foo: %w", WrapErrorTimeMessagef(nil, "earth-%s", "lings"))),
			)
			assertHasTimestamp(
				ErrorTimeFinder(WrapErrorTimeMessagef(fmt.Errorf("foo: %w", errors.New("hello")), "earth-%s", "lings")),
			)
			assertHasTimestamp(
				ErrorTimeFinder(fmt.Errorf("foo: %w", WrapErrorTimeMessagef(errors.New("hello"), "earth-%s", "lings"))),
			)
		})
	})
	t.Run("Interfaces", func(t *testing.T) {
		t.Run("Cause", func(t *testing.T) {
			we := WrapErrorTime(errors.New("hello"))
			if we.(interface{ Cause() error }).Cause().Error() != "hello" {
				t.Fatalf("error does not unwrap correctly: %v", we)
			}
		})
		t.Run("Cause", func(t *testing.T) {
			we := WrapErrorTime(errors.New("hello"))
			if we.(interface{ Unwrap() error }).Unwrap().Error() != "hello" {
				t.Fatalf("error does not unwrap correctly: %v", we)
			}
		})
		t.Run("FormattingBasic", func(t *testing.T) {
			err := &timestampError{
				time: time.Now(),
				err:  errors.New("hello world"),
			}
			if !strings.Contains(err.Error(), err.time.Format(time.RFC3339)) {
				t.Fatalf("timestamp is not present: %v", err)
			}
			if !strings.Contains(err.Error(), "hello world") {
				t.Fatalf("error string is not present: %v", err)
			}
		})
		t.Run("FormattingStrong", func(t *testing.T) {
			err := &timestampError{
				time: time.Now(),
				err:  errors.New("hello world"),
			}
			for _, fstr := range []string{"%+v", "%v", "%s", "%q"} {
				t.Run(fstr, func(t *testing.T) {
					if !strings.Contains(fmt.Sprintf("%+v", err), err.time.Format(time.RFC3339)) {
						t.Fatalf("timestamp is not present in [%s]: %v", fstr, err)
					}
					if !strings.Contains(fmt.Sprintf("%+v", err), "hello world") {
						t.Fatalf("error string is not present [%s]: %v", fstr, err)
					}
				})
			}
		})
	})
	t.Run("QuotedFormatting", func(t *testing.T) {
		err := newTimeStampError(fmt.Errorf("hello"))
		if strings.Contains(err.Error(), `"hello"`) {
			t.Fatalf("error produced in wrong form: [%v]", err)
		}
		if !strings.Contains(fmt.Sprintf("%q", err), `"hello"`) {
			t.Fatalf("error produced in wrong form: [%q]", err)
		}
	})
	t.Run("NillError", func(t *testing.T) {
		err := &timestampError{}
		if err.String() != "" {
			t.Fatal("string form of zero ts err should be empty")
		}
	})

	t.Run("NegativeCapacity", func(t *testing.T) {
		assertCapacityIsAtLeast(t, MakeTimestampCatcher(0), 0)
		assertCapacityIsAtLeast(t, MakeTimestampCatcher(1), 1)
		assertCapacityIsAtLeast(t, MakeTimestampCatcher(-1), 0)

		assertCapacityIsAtLeast(t, MakeExtendedTimestampCatcher(0), 0)
		assertCapacityIsAtLeast(t, MakeExtendedTimestampCatcher(1), 1)
		assertCapacityIsAtLeast(t, MakeExtendedTimestampCatcher(-1), 0)
	})

}
