package delaytimer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDelayTimer(t *testing.T) {
	dt := New(3)
	testdata := []struct {
		in     time.Duration
		expect time.Duration
	}{
		{time.Duration(1), time.Duration(0)},  // not included
		{time.Duration(3), time.Duration(2)},  // 3 - 1/1 = 2
		{time.Duration(8), time.Duration(6)},  // 8 - (3+1)/2 = 6
		{time.Duration(7), time.Duration(3)},  // 7 - (8+3+1)/3 = 3
		{time.Duration(3), time.Duration(0)},  // 3 - (7+8+3)/3 < 0
		{time.Duration(14), time.Duration(8)}, // 14 - (3+7+8)/3 = 8
	}
	for i, d := range testdata {
		assert.Equal(t, d.expect, dt.Calc(d.in), fmt.Sprintf("index: %d, ring: %v", i, dt.ring))
	}
}
