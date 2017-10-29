package retry

import (
	"errors"
	"math/rand"
	"time"
)

// ErrCanceled is used to cancel the retry process.
var ErrCanceled = errors.New("retry canceled")

// RetryFunc is a type which used for Retrier.
type RetryFunc func() error

// Retrier is a simple retry implemention.
type Retrier struct {
	backoff time.Duration
	max     int
}

// New creates a Retrier, the backoff increment is not exponential,
// it is a random value.
func New(backoff time.Duration, max int) *Retrier {
	return &Retrier{
		backoff: backoff,
		max:     max,
	}
}

// Run runs the RetryFunc until success or excceed the max retry times.
func (r *Retrier) Run(fn RetryFunc) (err error) {
	backoff := r.backoff

	for i, max := 1, r.max+1; i < max; i++ {
		if err = fn(); err == nil {
			return
		}
		if err == ErrCanceled {
			return
		}
		time.Sleep(backoff)

		backoff = backoff*time.Duration(i) + time.Duration(float64(backoff)*rand.Float64())
	}
	return
}

// Retry creates a shortcut retry function for later easier reuse.
func Retry(fn RetryFunc, backoff time.Duration, max int) func() error {
	r := New(backoff, max)
	return func() error {
		return r.Run(fn)
	}
}
