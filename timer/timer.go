package timer

import (
	"time"
)

type (
	// Timer is an interface which implements all time-related
	// functions. It can be mocked for testing.
	Timer interface {
		// Now returns current time.
		Now() time.Time
		// Reset
		Reset(s HV, d time.Duration)
		// Sleep stops execution for duration d.
		Sleep(d time.Duration)
		// Extend extends current timer with duration d.
		Extend(d time.Duration)
		// Stop stops timer.
		Stop()
		// HV returns current height and view set for the timer.
		HV() HV
		// C returns channel for timer events.
		C() <-chan time.Time
	}

	value struct {
		HV
		s time.Time
		d time.Duration
	}

	// HV is a pair of a Height and a View.
	HV struct {
		Height uint32
		View   byte
	}

	timer struct {
		val value
		tt  *time.Timer
		ch  chan time.Time
	}
)

var _ Timer = (*timer)(nil)

// New returns default Timer implementation.
func New() Timer {
	t := &timer{
		ch: make(chan time.Time, 1),
	}

	return t
}

// C implements Timer interface.
func (t *timer) C() <-chan time.Time {
	if t.tt == nil {
		return t.ch
	}

	return t.tt.C
}

// HV implements Timer interface.
func (t *timer) HV() HV {
	return t.val.HV
}

// Reset implements Timer interface.
func (t *timer) Reset(hv HV, d time.Duration) {
	t.Stop()

	t.val.s = t.Now()
	t.val.d = d
	t.val.HV = hv

	if t.val.d != 0 {
		t.tt = time.NewTimer(t.val.d)
	} else {
		t.tt = nil
		drain(t.ch)
		t.ch <- t.val.s
	}
}

func drain(ch <-chan time.Time) {
	select {
	case <-ch:
	default:
	}
}

// Stop implements Timer interface.
func (t *timer) Stop() {
	if t.tt != nil {
		t.tt.Stop()
		t.tt = nil
	}
}

// Sleep implements Timer interface.
func (t *timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Extend implements Timer interface.
func (t *timer) Extend(d time.Duration) {
	t.val.d += d

	if elapsed := time.Since(t.val.s); t.val.d > elapsed {
		t.Stop()
		t.tt = time.NewTimer(t.val.d - elapsed)
	}
}

// Now implements Timer interface.
func (t *timer) Now() time.Time {
	return time.Now()
}
