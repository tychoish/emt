// Error Channel
//
// The ErrorChannel implementation provides a safe interface for
// collecting and managing errors in situations that involve channels,
// and provides a bridge between Catchers and more event-driven
// cases.
//
// This implementation wraps a catcher to collect errors separate from
// the channel interface which is managed by a goroutine that
// processes errors between the "IN" and "OUT" channels, which are
// buffered independently.
package emt

import "context"

// ErrorChannel provides an error management utility for integration
// in code that makes use of channels.
type ErrorChannel struct {
	errRecv chan error
	errSend chan error
	catcher Catcher
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewErrorChannel constructs and starts an ErrorChannel instance. The
// size controls the buffer on the input and output channels, which
// are buffered separately. A size of 32 will result in an object
// which can store 64 errors in the channels, although the embedded
// Catcher will store *all* submitted errors.
func NewErrorChannel(ctx context.Context, size int) *ErrorChannel {
	ec := &ErrorChannel{
		errRecv: make(chan error, size),
		errSend: make(chan error, size),
		catcher: NewCatcher(),
	}
	ec.ctx, ec.cancel = context.WithCancel(ctx)
	go ec.start(ec.ctx)

	return ec
}

func (ec *ErrorChannel) start(ctx context.Context) {
	defer func() {
		r := recover()
		if r != nil {
			ec.catcher.Errorf("channel processor encountered panic: %v", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-ec.errRecv:
			if err == nil {
				continue
			}
			ec.catcher.Add(err)
			select {
			case <-ctx.Done():
				return
			case ec.errSend <- err:
			}
		}
	}

}

// Stop aborts the background process that handles errors, and will
// cause the Wait method to return the resolved errors collected by
// the object over it's lifetime.
func (ec *ErrorChannel) Stop() { ec.cancel() }

// Resolve returns an aggregated error observed by the ErrorChannel.
func (ec *ErrorChannel) Resolve() error { return ec.catcher.Resolve() }

// In returns a channel that you can use to submit errors to the
// collector. This channel is never closed.
func (ec *ErrorChannel) In() chan<- error { return ec.errRecv }

// In returns a channel that you can use to consume errors from the
// error channel. This channel is never closed.
func (ec *ErrorChannel) Out() <-chan error { return ec.errSend }

// Collect saves the error in question in the underlying Catcher and
// then sends the error to the OUT channel, blocking until that
// channel has capacity, Stop is called or the context is canceled. If
// the context is canceled or Stop is called, the error may not be
// propogated to the Out channel.
//
// All errors are saved to the catcher regardless of the state of the
// context, ErrorChannel's background thread or the OUT channel.
func (ec *ErrorChannel) Collect(ctx context.Context, err error) {
	if err != nil {
		ec.catcher.Add(err)
		select {
		case <-ctx.Done():
		case <-ec.ctx.Done():
		case ec.errSend <- err:
		}
	}
}

// Wait blocks until the context is canceled, returning a
// context.Canceled or context.DeadlineExceeded error (typically) or
// the Stop() method is called, and then returns a resolved error from
// the Catcher instance that's collected all errors.
func (ec *ErrorChannel) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ec.ctx.Done():
		return ec.Resolve()
	}
}
