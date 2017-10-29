package delaytimer

import "time"

// DelayTimer calculates delayed duration,
// it use current duration compares with
// the average duration of latest n duration.
type DelayTimer struct {
	ring  []time.Duration
	next  int
	count int
	num   int
	sum   time.Duration
}

// New creates a new DelayTimer.
func New(num int) *DelayTimer {
	return &DelayTimer{
		ring:  make([]time.Duration, num, num),
		next:  0,
		count: 0,
		num:   num,
		sum:   time.Duration(0),
	}
}

// Calc calculates calculates delay then append current duration into cache,
// always return 0 if delay duration less than 0.
func (dc *DelayTimer) Calc(d time.Duration) time.Duration {
	defer dc.add(d)

	if dc.count > 0 {
		delayed := d - (dc.sum / time.Duration(dc.count))
		if delayed > 0 {
			return delayed
		}
	}
	return time.Duration(0)
}

func (dc *DelayTimer) add(in time.Duration) {
	var out time.Duration
	if dc.count < dc.num {
		dc.ring[dc.next] = in
		dc.count++
	} else {
		out = dc.ring[dc.next]
		dc.ring[dc.next] = in
	}
	dc.sum += in - out
	dc.next = (dc.next + 1) % dc.num
}
