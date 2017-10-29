package retry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	cnt := 0
	retryFunc := func() error {
		cnt++
		return fmt.Errorf("rt")
	}

	backoff := time.Millisecond
	now := time.Now()
	err := Retry(retryFunc, backoff, 5)()
	assert.NotNil(t, err)
	assert.Equal(t, 5, cnt)
	assert.True(t, time.Now().Sub(now) > backoff*10)

	cnt = 0
	retryFunc = func() error {
		cnt++
		return nil
	}

	now = time.Now()
	err = Retry(retryFunc, backoff, 5)()
	assert.Nil(t, err)
	assert.Equal(t, 1, cnt)
	assert.True(t, time.Now().Sub(now) < backoff)
}
