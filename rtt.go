package dbft

import (
	"time"
)

const rttLength = 7 * 10 // 10 rounds with 7 nodes

type rtt struct {
	times [rttLength]time.Duration
	idx   int
	avg   time.Duration
}

func (r *rtt) addTime(t time.Duration) {
	var old = r.times[r.idx]

	if old != 0 {
		t = min(t, 2*old) // Too long delays should be normalized, we don't want to overshoot.
	}

	r.avg = r.avg + (t-old)/time.Duration(len(r.times))
	r.avg = max(0, r.avg) // Can't be less than zero.
	r.times[r.idx] = t
	r.idx = (r.idx + 1) % len(r.times)
}
