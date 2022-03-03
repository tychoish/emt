package emt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestCatcher(t *testing.T) {
	type fixture struct {
		Name      string
		Factory   func() Catcher
		FixedSize int
	}
	fixtures := []fixture{
		{
			Name:    "Catcher",
			Factory: NewCatcher,
		},
		{
			Name:    "ExtendedCatcher",
			Factory: NewExtendedCatcher,
		},
		{
			Name:    "BasicCatcher",
			Factory: NewBasicCatcher,
		},
		{
			Name:    "Simple",
			Factory: NewSimpleCatcher,
		},
		{
			Name:    "Timestamp",
			Factory: NewTimestampCatcher,
		},
		{
			Name:    "ExtendedTimestamp",
			Factory: NewExtendedCatcher,
		},
	}

	for _, size := range []int{10, 100, 1000} {
		fixtures = append(fixtures,
			fixture{
				Name:      fmt.Sprintf("Fixed/Extended/%d", size),
				Factory:   func() Catcher { return MakeExtendedCatcher(size) },
				FixedSize: size,
			},
			fixture{
				Name:      fmt.Sprintf("Fixed/Basic/%d", size),
				Factory:   func() Catcher { return MakeBasicCatcher(size) },
				FixedSize: size,
			},
			fixture{
				Name:      fmt.Sprintf("Fixed/Simple/%d", size),
				Factory:   func() Catcher { return MakeSimpleCatcher(size) },
				FixedSize: size,
			},
			fixture{
				Name:      fmt.Sprintf("Fixed/Timestamp/%d", size),
				Factory:   func() Catcher { return MakeTimestampCatcher(size) },
				FixedSize: size,
			},
			fixture{
				Name:      fmt.Sprintf("Fixed/ExtendedTimestamp/%d", size),
				Factory:   func() Catcher { return MakeExtendedTimestampCatcher(size) },
				FixedSize: size,
			},
		)
	}

	testCases := []struct {
		Name string
		Case func(t *testing.T, catcher Catcher, size int)
	}{
		{
			Name: "InitialValues",
			Case: func(t *testing.T, catcher Catcher, size int) {
				if catcher.HasErrors() {
					t.Fatal("catcher should not have errors upon creation")
				}
				if catcher.Len() != 0 {
					t.Fatal("new catchers should have zero length")
				}
				if catcher.String() != "" {
					t.Fatal("catchers should not resolve a string")
				}
				if catcher.Resolve() != nil {
					t.Fatal("catchers should not resolve an error")
				}
			},
		},
		{
			Name: "AddMethodImpactsState",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)

				catcher.Add(errors.New("foo"))

				assertCatcherHasErrors(t, catcher, 1)
			},
		},
		{
			Name: "AddingNilMethodDoesNotImpactCatcherState",
			Case: func(t *testing.T, catcher Catcher, size int) {
				for i := 0; i < 100; i++ {
					catcher.Add(nil)
				}
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "AddingManyErrorsIsCaptured",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 100)

				for i := 1; i <= 100; i++ {
					catcher.Add(errors.New(strconv.Itoa(i)))
					assertCatcherHasErrors(t, catcher, i)
				}

				assertCatcherHasErrors(t, catcher, 100)
			},
		},
		{
			Name: "ResolveMethodIsNilIfNotHasErrors",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCatcherEmpty(t, catcher)

				for i := 0; i < 100; i++ {
					catcher.Add(nil)
					assertCatcherEmpty(t, catcher)
				}

				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "CheckExtendNoError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				fn := func() error { return nil }
				catcher.CheckExtend([]CheckFunction{fn, fn, fn})

				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "CheckWhenNoError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				fn := func() error { return nil }

				catcher.CheckWhen(false, fn)

				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "ResolveMethodDoesNotClearStateOfCatcher",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 10)

				assertCatcherEmpty(t, catcher)

				for i := 1; i <= 10; i++ {
					catcher.Add(errors.New(strconv.Itoa(i)))
					assertCatcherHasErrors(t, catcher, i)
				}

				assertCatcherHasErrors(t, catcher, 10)
			},
		},
		{
			Name: "ConcurrentAdd",
			Case: func(t *testing.T, catcher Catcher, size int) {
				batchSize := 256
				if capper, ok := catcher.(interface{ Cap() int }); ok {
					if capper.Cap() > batchSize {
						batchSize = capper.Cap()
					}
				}

				assertCatcherEmpty(t, catcher)
				wg := &sync.WaitGroup{}
				wg.Add(batchSize)
				for i := 0; i < batchSize; i++ {
					go func(num int) {
						catcher.Add(fmt.Errorf("adding err #%d", num))
						wg.Done()
					}(i)
				}
				wg.Wait()
				assertCatcherHasErrors(t, catcher, batchSize)
			},
		},
		{
			Name: "ErrorsAndExtend",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 20)

				for i := 1; i <= 10; i++ {
					catcher.Add(errors.New(strconv.Itoa(i)))
					assertCatcherHasErrors(t, catcher, i)
				}

				assertCatcherHasErrors(t, catcher, 10)

				errs := catcher.Errors()
				if len(errs) != 10 {
					t.Fatalf("there are %d errors and there should only be 10", len(errs))
				}

				catcher.Extend(errs)
				assertCatcherHasErrors(t, catcher, 20)
			},
		},
		{
			Name: "ExtendWithEmptySet",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCatcherEmpty(t, catcher)
				catcher.Extend(catcher.Errors())
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "ExtendWithNilErrors",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)
				errs := []error{nil, errors.New("what"), nil}
				catcher.Extend(errs)
				if l := catcher.Len(); l != 1 {
					t.Fatalf("catcher has %d", l)
				}
			},
		},
		{
			Name: "AddWhenNilError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.AddWhen(true, nil)
				assertCatcherEmpty(t, catcher)

				catcher.AddWhen(false, nil)
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "NewEmpty",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.New("")
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "NewAddsAndPropogated",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)
				catcher.New("one")
				assertCatcherHasErrors(t, catcher, 1)
				if !strings.Contains(catcher.Error(), "one") {
					t.Fatalf("error is not propagated: [%q], %v", "one", catcher.Error())
				}
			},
		},
		{
			Name: "AddWithErrorAndFalse",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)

				catcher.AddWhen(false, errors.New("f"))
				assertCatcherEmpty(t, catcher)

				catcher.AddWhen(true, errors.New("f"))
				assertCatcherHasErrors(t, catcher, 1)

				catcher.AddWhen(false, errors.New("f"))
				assertCatcherHasErrors(t, catcher, 1)
			},
		},
		{
			Name: "ExtendWhenNilError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.ExtendWhen(true, []error{})
				assertCatcherEmpty(t, catcher)

				catcher.ExtendWhen(false, []error{})
				assertCatcherEmpty(t, catcher)

				catcher.ExtendWhen(true, nil)
				assertCatcherEmpty(t, catcher)

				catcher.ExtendWhen(false, nil)
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "ExtendWithErrorAndFalse",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 3)

				catcher.ExtendWhen(false, []error{errors.New("f")})
				assertCatcherEmpty(t, catcher)

				catcher.ExtendWhen(true, []error{errors.New("f")})
				assertCatcherHasErrors(t, catcher, 1)

				catcher.ExtendWhen(false, []error{errors.New("f")})
				assertCatcherHasErrors(t, catcher, 1)

				catcher.ExtendWhen(false, []error{errors.New("f"), errors.New("f")})
				assertCatcherHasErrors(t, catcher, 1)

				catcher.ExtendWhen(true, []error{errors.New("f"), errors.New("f")})
				assertCatcherHasErrors(t, catcher, 3)
			},
		},
		{
			Name: "ErrorfNilCases",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.Errorf("")
				assertCatcherEmpty(t, catcher)

				catcher.Errorf("", true, false)
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "NewWhen",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)

				catcher.NewWhen(false, "")
				assertCatcherEmpty(t, catcher)

				catcher.NewWhen(false, "one")
				assertCatcherEmpty(t, catcher)

				catcher.NewWhen(true, "one")
				assertCatcherHasErrors(t, catcher, 1)

				catcher.NewWhen(true, "")
				assertCatcherHasErrors(t, catcher, 1)

				catcher.NewWhen(false, "one")
				assertCatcherHasErrors(t, catcher, 1)
			},
		},
		{
			Name: "ErrorfWhenNilCases",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.ErrorfWhen(true, "")
				assertCatcherEmpty(t, catcher)

				catcher.ErrorfWhen(true, "", true, false)
				assertCatcherEmpty(t, catcher)

				catcher.ErrorfWhen(false, "")
				assertCatcherEmpty(t, catcher)

				catcher.ErrorfWhen(false, "", true, false)
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "ErrorfNoArgs",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)
				catcher.Errorf("%s what")
				assertCatcherHasErrors(t, catcher, 1)

				if !strings.Contains(catcher.Error(), "%s what") {
					t.Fatalf("error is not properly propagated: %v", catcher.Error())
				}
			},
		},
		{
			Name: "CheckWhen",
			Case: func(t *testing.T, catcher Catcher, size int) {
				fn := func() error { return nil }

				catcher.CheckWhen(false, fn)
				assertCatcherEmpty(t, catcher)

				catcher.CheckWhen(true, fn)
				assertCatcherEmpty(t, catcher)

				catcher.Check(fn)
				assertCatcherEmpty(t, catcher)
			},
		},
		{
			Name: "ErrorfWhenNoArgs",
			Case: func(t *testing.T, catcher Catcher, size int) {
				catcher.ErrorfWhen(false, "%s what")
				assertCatcherEmpty(t, catcher)

				catcher.ErrorfWhen(true, "%s what")
				assertCatcherHasErrors(t, catcher, 1)

				if !strings.Contains(catcher.Error(), "%s what") {
					t.Fatalf("error is not propagated: %v", catcher.Error())
				}
			},
		},
		{
			Name: "ErrorfFull",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)
				catcher.Errorf("%s what", "this")

				assertCatcherHasErrors(t, catcher, 1)
				if !strings.Contains(catcher.Error(), "this what") {
					t.Fatalf("error is not propagated: %v", catcher.Error())
				}
			},
		},
		{
			Name: "WhenErrorfFull",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)

				catcher.ErrorfWhen(false, "%s what", "this")
				assertCatcherEmpty(t, catcher)

				catcher.ErrorfWhen(true, "%s what", "this")
				assertCatcherHasErrors(t, catcher, 1)
				if !strings.Contains(catcher.Error(), "this what") {
					t.Fatalf("error is not propagated: %v", catcher.Error())
				}
			},
		},
		{
			Name: "CheckWhenError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 1)

				fn := func() error { return errors.New("hi") }
				if err := fn(); err == nil {
					t.Fatal("problem with error fixture")
				}

				catcher.CheckWhen(false, fn)
				assertCatcherEmpty(t, catcher)

				catcher.CheckWhen(true, fn)
				assertCatcherHasErrors(t, catcher, 1)

				catcher.Check(fn)
				assertCatcherHasErrors(t, catcher, 2)
			},
		},
		{
			Name: "CheckExtendError",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 3)

				fn := func() error { return errors.New("hi") }
				catcher.CheckExtend([]CheckFunction{fn, fn, fn})

				assertCatcherHasErrors(t, catcher, 3)

			},
		},
		{
			Name: "CheckExtendMixed",
			Case: func(t *testing.T, catcher Catcher, size int) {
				assertCapacityIsAtLeast(t, catcher, 2)

				catcher.CheckExtend([]CheckFunction{
					func() error { return errors.New("hi") },
					func() error { return errors.New("hi") },
					func() error { return nil },
				})

				assertCatcherHasErrors(t, catcher, 2)
			},
		},
		{
			Name: "CapsRespectedForAdd",
			Case: func(t *testing.T, catcher Catcher, size int) {
				if size <= 0 {
					size = 256
				}

				for i := 0; i < 2*size; i++ {
					catcher.Add(errors.New("abc"))
				}

				if size == 0 {
					assertCatcherHasErrors(t, catcher, 2*size)
				} else {
					assertCatcherHasErrors(t, catcher, size)
				}
			},
		},
		{
			Name: "CapsRespectedForExtend",
			Case: func(t *testing.T, catcher Catcher, size int) {
				if size <= 0 {
					size = 256
				}

				errs := make([]error, size*2)

				for idx := range errs {
					errs[idx] = errors.New("abc")
				}

				catcher.Extend(errs)
				if size == 0 {
					assertCatcherHasErrors(t, catcher, 2*size)
				} else {
					assertCatcherHasErrors(t, catcher, size)
				}

				catcher.Extend(errs)

				if size == 0 {
					assertCatcherHasErrors(t, catcher, 4*size)
				} else {
					assertCatcherHasErrors(t, catcher, size)
				}

			},
		},
	}

	for _, fix := range fixtures {
		t.Run(fix.Name, func(t *testing.T) {
			if fix.Factory() == nil {
				t.Fatal("invalid factory")
			}

			for _, test := range testCases {
				if test.Name == "" {
					continue
				}
				t.Run(test.Name, func(t *testing.T) {
					catcher := fix.Factory()
					if fix.FixedSize > 0 {
						assertCapacityIsAtLeast(t, catcher, fix.FixedSize)
					}
					test.Case(t, catcher, fix.FixedSize)
				})
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////

func assertCatcherEmpty(t *testing.T, catcher Catcher) {
	t.Helper()
	var shouldError bool
	if err := catcher.Resolve(); err != nil {
		shouldError = true
		t.Errorf("catcher returned an error, but shouldn't have: %v", err)
	}
	if l := catcher.Len(); l != 0 {
		shouldError = true
		t.Errorf("catcher has %d length, but should have 0", l)
	}
	if catcher.HasErrors() {
		shouldError = true
		t.Error("catcher reports having errors and it should not")
	}

	if shouldError {
		t.FailNow()
	}
}

func assertCatcherHasErrors(t *testing.T, catcher Catcher, size int) {
	t.Helper()
	var shouldError bool
	if !catcher.HasErrors() {
		shouldError = true
		t.Error("catcher reports having errors and it shouldn't")
	}
	if err := catcher.Resolve(); err == nil {
		shouldError = true
		t.Error("catcher did not return an error bu should have")
	}

	if l := catcher.Len(); l < size {
		shouldError = true
		t.Errorf("catcher has length of %d but should have %d", l, size)
	}

	if shouldError {
		t.FailNow()
	}
}

func assertCapacityIsAtLeast(t *testing.T, catcher Catcher, size int) {
	if size <= 0 {
		return
	}

	t.Helper()

	capper, ok := catcher.(interface{ Cap() int })
	if !ok {
		t.Fatalf("%T does not report capacity", catcher)
	}
	if cp := capper.Cap(); cp != 0 && cp < size {
		t.Fatalf("cp=%d, which is not greater than target %d", cp, size)
	}
}
