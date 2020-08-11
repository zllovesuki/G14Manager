package util

import (
	"time"
)

// DebounceEvent contains the last event fired to the input channel
type DebounceEvent struct {
	Counter int64
	Data    interface{}
}

// Debounce returns two channels for (dirty) input and (clean) output.
func Debounce(wait time.Duration) (chan<- interface{}, <-chan DebounceEvent) {
	in := make(chan interface{})
	out := make(chan DebounceEvent, 1) // do not block our goroutine

	go func(period time.Duration, listen <-chan interface{}, release chan<- DebounceEvent) {
		var counter int64
		var data interface{}
		var timer <-chan time.Time

		for {
			select {
			case data = <-listen:
				timer = time.After(period)
				counter++
			case <-timer:
				release <- DebounceEvent{
					Counter: counter,
					Data:    data,
				}
				timer = nil
				counter = 0
			}
		}
	}(wait, in, out)

	return in, out
}
