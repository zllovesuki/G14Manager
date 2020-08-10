package util

import (
	"time"
)

// DebounceEvent contains the last event fired to the input channel
type DebounceEvent struct {
	Counter int64
	Data    interface{}
}

// Debounce returns two channels for input and output.
func Debounce(wait time.Duration) (noisy chan interface{}, clean chan DebounceEvent) {
	noisy = make(chan interface{})
	clean = make(chan DebounceEvent, 1) // do not block our goroutine

	go func() {
		var lastTime time.Time
		var counter int64
		var data interface{}

		for {
			select {
			case data = <-noisy:
				lastTime = time.Now()
				counter++
			case <-time.Tick(wait):
				if !lastTime.IsZero() && time.Now().Sub(lastTime) > wait {
					clean <- DebounceEvent{
						Counter: counter,
						Data:    data,
					}

					lastTime = time.Time{}
					counter = 0
				}
			}
		}
	}()

	return
}
