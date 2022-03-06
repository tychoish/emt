package emt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	fixtures := []struct {
		Name    string
		Factory func(ctx context.Context, size int) *ErrorChannel
	}{
		{
			Name: "Basic",
			Factory: func(ctx context.Context, size int) *ErrorChannel {
				return NewErrorChannel(ctx, size)
			},
		},
	}

	cases := []struct {
		Name string
		Test func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int)
	}{
		{
			Name: "Validate",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				if ec == nil {
					t.Fatal("fixture should produce non-nil ")

				}
				if size < 0 {
					t.Fatal("cases must define valid size")
				}
			},
		},
		{
			Name: "NilError",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				ec.Collect(ctx, nil)
				if ec.Resolve() != nil {
					t.Fatal("collecting a nil error should not result in an error")
				}
			},
		},
		{
			Name: "SimpleCollect",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				ec.Collect(ctx, errors.New("hi"))
				if err := ec.Resolve(); err == nil {
					t.Fatal("collecting an error should result in a resolved error")
				} else if err.Error() != "hi" {
					t.Fatal("error is not correctly propagated")
				}

			},
		},
		{
			Name: "PanicSafety",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				// this will cause it to panic eventually panic eventually
				ec.start(nil)

				if err := ec.Resolve(); err == nil {
					t.Fatal("internal panic not recorded")
				} else if !strings.Contains(err.Error(), "encountered panic") {
					t.Fatalf("internal panic not captured: %v", err)
				}
			},
		},
		{
			Name: "NilSendsDoNotResolve",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				for i := 0; i < size; i++ {
					ec.In() <- nil
				}
				if err := ec.Resolve(); err != nil {
					t.Fatalf("nil sends should not return an error: %v", err)
				}

			},
		},
		{
			Name: "WaitRespectsContext",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				stopctx, cancel := context.WithCancel(ctx)
				cancel()
				if err := ec.Wait(stopctx); !errors.Is(err, context.Canceled) {
					t.Fatalf("context cancelation should be propogated: %v", err)
				}
			},
		},
		{
			Name: "WaitRespectsStop",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				ec.Stop()
				if err := ec.Wait(ctx); err != nil {
					t.Fatalf("should not report errors for normal stop, %v", err)
				}
			},
		},
		{
			Name: "WaitPropogatesErrorValue",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				ec.Collect(ctx, errors.New("hi"))
				ec.Stop()
				if err := ec.Wait(ctx); err == nil || err.Error() != "hi" {
					t.Fatalf("should not report errors for normal stop, %v", err)
				}
			},
		},
		{
			Name: "ErrorsGoToOut",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				ec.Collect(ctx, errors.New("hi"))
				err, _ := <-ec.Out()
				if err == nil {
					t.Fatal("no error propogated")
				}
				if err.Error() != "hi" {
					t.Fatalf("incorrect error value: %v", err)
				}
			},
		},
		{
			Name: "NoPropogationAfterStop",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 4*time.Millisecond)
				defer cancel()

				send := ec.In()
				ec.Stop()
				time.Sleep(time.Millisecond)
				go func() {
					count := 0
					for {
						select {
						case <-ctx.Done():
							if count == 0 {
								t.Errorf("should have produced at least one error [%d]", count)
							}
							return
						case send <- errors.New("hi"):
							count++
						}
					}
				}()

				select {
				case <-ctx.Done():
					if ec.catcher.HasErrors() {
						t.Fatal("catcher should not have errors")
					}
				case err := <-ec.Out():
					if err == nil {
						return
					}
					t.Fatalf("produced error but should not have: %v", err)
				}

			},
		},
		{
			Name: "Propogation",
			Test: func(ctx context.Context, t *testing.T, ec *ErrorChannel, size int) {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Millisecond)
				defer cancel()

				send := ec.In()
				go func() {
					count := 0
					for {
						select {
						case <-ctx.Done():
							if count == 0 {
								t.Errorf("should have produced at least one error [%d]", count)
							}
							return
						case send <- errors.New("hi"):
							count++
						}
					}
				}()

				select {
				case <-ctx.Done():
					if !ec.catcher.HasErrors() {
						t.Fatal("catcher should not have errors")
					}
				case err := <-ec.Out():
					if err == nil {
						t.Fatal("did not produce error but should  have")
					}
				}

			},
		},
	}

	for _, fix := range fixtures {
		t.Run(fix.Name, func(t *testing.T) {
			for _, size := range []int{1, 4, 8, 16, 32, 64, 128} {
				t.Run(fmt.Sprint(size), func(t *testing.T) {
					for _, test := range cases {
						t.Run(test.Name, func(t *testing.T) {
							ctx, cancel := context.WithCancel(context.Background())
							defer cancel()
							ec := fix.Factory(ctx, size)

							test.Test(ctx, t, ec, size)
						})
					}
				})
			}
		})
	}
}
