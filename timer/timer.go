/*
Package timer contains default implementation of [dbft.Timer] interface and provides
all necessary timer-related functionality to [dbft.DBFT] service.
*/
package timer

import (
	"time"
)

type (
	// Timer is a default [dbft.Timer] implementation.
	Timer struct {
		height uint32
		view   byte
		s      time.Time
		d      time.Duration
		tt     *time.Timer
		ch     chan time.Time
	}
)

// New returns default Timer implementation.
func New() *Timer {
	t := &Timer{
		ch: make(chan time.Time, 1),
	}

	return t
}

// C implements Timer interface.
func (t *Timer) C() <-chan time.Time {
	if t.tt == nil {
		return t.ch
	}

	return t.tt.C
}

// Height returns current timer height.
func (t *Timer) Height() uint32 {
	return t.height
}

// View return current timer view.
func (t *Timer) View() byte {
	return t.view
}

// Reset implements Timer interface.
func (t *Timer) Reset(height uint32, view byte, d time.Duration) {
	t.Stop()

	t.s = t.Now()
	t.d = d
	t.height = height
	t.view = view

	if t.d != 0 {
		t.tt = time.NewTimer(t.d)
	} else {
		t.tt = nil
		drain(t.ch)
		t.ch <- t.s
	}
}

func drain(ch <-chan time.Time) {
	select {
	case <-ch:
	default:
	}
}

// Stop implements Timer interface.
func (t *Timer) Stop() {
	if t.tt != nil {
		t.tt.Stop()
		t.tt = nil
	}
}

// Sleep implements Timer interface.
func (t *Timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Extend implements Timer interface.
func (t *Timer) Extend(d time.Duration) {
	t.d += d

	if elapsed := time.Since(t.s); t.d > elapsed {
		t.Stop()
		t.tt = time.NewTimer(t.d - elapsed)
	}
}

// Now implements Timer interface.
func (t *Timer) Now() time.Time {
	return time.Now()
}
