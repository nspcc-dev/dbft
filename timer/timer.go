/*
Package timer contains default implementation of [dbft.Timer] interface and provides
all necessary timer-related functionality to [dbft.DBFT] service.
*/
package timer

import (
	"time"

	"github.com/nspcc-dev/dbft"
)

type (
	value struct {
		HV
		s time.Time
		d time.Duration
	}

	// HV is a pair of a H and a V that implements [dbft.HV] interface.
	HV struct {
		H uint32
		V byte
	}

	// Timer is a default [dbft.Timer] implementation.
	Timer struct {
		val value
		tt  *time.Timer
		ch  chan time.Time
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

// HV implements Timer interface.
func (t *Timer) HV() dbft.HV {
	return t.val.HV
}

// Reset implements Timer interface.
func (t *Timer) Reset(hv dbft.HV, d time.Duration) {
	t.Stop()

	t.val.s = t.Now()
	t.val.d = d
	t.val.HV = HV{
		H: hv.Height(),
		V: hv.View(),
	}

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
	t.val.d += d

	if elapsed := time.Since(t.val.s); t.val.d > elapsed {
		t.Stop()
		t.tt = time.NewTimer(t.val.d - elapsed)
	}
}

// Now implements Timer interface.
func (t *Timer) Now() time.Time {
	return time.Now()
}

// NewHV is a constructor of HV.
func NewHV(height uint32, view byte) dbft.HV {
	return HV{
		H: height,
		V: view,
	}
}

// Height implements [dbft.HV] interface.
func (hv HV) Height() uint32 {
	return hv.H
}

// View implements [dbft.HV] interface.
func (hv HV) View() byte {
	return hv.V
}
