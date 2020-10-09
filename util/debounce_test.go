package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDebounce(t *testing.T) { // -race passes
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	in, out := Debounce(context.Background(), time.Millisecond*25)
	rounds := int64(10)

	go func() {
		numReceived := 0
		var lastReceived DebounceEvent
		for {
			select {
			case i := <-out:
				lastReceived = i
				numReceived++
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					// only check after timeout
					t.Log("hello")
					require.Equal(t, 1, numReceived)
					i := lastReceived.Data.(int64)
					// off by one
					require.Equal(t, rounds, i)
					require.Equal(t, rounds, lastReceived.Counter)
				}
				return
			}
		}
	}()

	for i := int64(1); i <= rounds; i++ {
		in <- i
		time.Sleep(time.Millisecond * 5)
	}

	time.Sleep(time.Millisecond * 200)
}

func TestMultipleDebounce(t *testing.T) { // -race passes
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	in, out := Debounce(context.Background(), time.Millisecond*10)

	go func() {
		numReceived := 0
		lastReceived := make([]DebounceEvent, 0, 2)

		for {
			select {
			case ev := <-out:
				lastReceived = append(lastReceived, ev)
				numReceived++
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					// only check after timeout
					t.Log("hello again")
					require.Equal(t, 2, numReceived)
					require.Equal(t, "A", lastReceived[0].Data.(string))
					require.Equal(t, "B", lastReceived[1].Data.(string))
				}
				return
			}
		}
	}()

	in <- "A"
	time.Sleep(time.Millisecond * 50)
	in <- "B"

	time.Sleep(time.Millisecond * 200)

}
