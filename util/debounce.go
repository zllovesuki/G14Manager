package util

import (
	"context"
	"time"
)

// DebounceEvent contains the last event fired to the input channel
type DebounceEvent struct {
	Counter int64
	Data    interface{}
}

// Debounce returns two channels for (dirty) input and (clean) output.
func Debounce(haltCtx context.Context, wait time.Duration) (chan<- interface{}, <-chan DebounceEvent) {
	in := make(chan interface{})
	out := make(chan DebounceEvent, 1) // do not block our goroutine

	go func() {
		var counter int64
		var data interface{}
		var timer <-chan time.Time

		for {
			select {
			case data = <-in:
				timer = time.After(wait)
				counter++
			case <-timer:
				out <- DebounceEvent{
					Counter: counter,
					Data:    data,
				}
				timer = nil
				counter = 0
			case <-haltCtx.Done():
				return
			}
		}
	}()

	return in, out
}

// PassThrough will pipe (dirty) input directly to (clean) output without debouncing.
func PassThrough(haltCtx context.Context) (chan<- interface{}, <-chan DebounceEvent) {
	in := make(chan interface{})
	out := make(chan DebounceEvent, 1) // do not block our goroutine

	go func() {
		for {
			select {
			case data := <-in:
				out <- DebounceEvent{
					Counter: 1,
					Data:    data,
				}
			case <-haltCtx.Done():
				return
			}
		}
	}()

	return in, out
}
