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
		// C returns channel where HV events are arrived
		// after timer has fired.
		C() <-chan HV
	}

	value struct {
		HV
		s time.Time
		d time.Duration
		e bool
	}

	// HV is a pair of a Height and a View.
	HV struct {
		Height uint32
		View   byte
	}

	timer struct {
		ch     chan HV
		values chan value
		stop   chan struct{}
	}
)

var _ Timer = (*timer)(nil)

// New returns default Timer implementation.
func New() Timer {
	t := &timer{
		ch:     make(chan HV, 1),
		values: make(chan value),
		stop:   make(chan struct{}, 1),
	}

	go t.loop()

	return t
}

// C implements Timer interface.
func (t *timer) C() <-chan HV { return (<-chan HV)(t.ch) }

// Reset implements Timer interface.
func (t *timer) Reset(hv HV, d time.Duration) {
	t.values <- value{
		HV: hv,
		s:  t.Now(),
		d:  d,
	}
}

// Stop implements Timer interface.
func (t *timer) Stop() {
	close(t.stop)
}

// Sleep implements Timer interface.
func (t *timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

func getChan(tt *time.Timer) <-chan time.Time {
	if tt == nil {
		return nil
	}

	return tt.C
}

func stopTimer(tt *time.Timer) {
	if tt != nil {
		tt.Stop()
	}
}

func drain(ch <-chan HV) {
	select {
	case <-ch:
	default:
	}
}

func (t *timer) loop() {
	var tt *time.Timer
	var toSend value

	for {
		select {
		case v := <-t.values:
			if !v.e {
				toSend.HV = v.HV
				toSend.s = v.s
				toSend.d = v.d
			} else {
				toSend.d *= v.d
			}

			stopTimer(tt)

			elapsed := time.Since(toSend.s)
			tt = time.NewTimer(toSend.d - elapsed)

		case <-getChan(tt):
			stopTimer(tt)
			tt = nil

			drain(t.ch)
			t.ch <- toSend.HV

		case _, ok := <-t.stop:
			stopTimer(tt)
			tt = nil

			if !ok {
				drain(t.ch)
				return
			}
		}
	}
}

// Extend implements Timer interface.
func (t *timer) Extend(d time.Duration) {
	t.values <- value{d: d, e: true}
}

// Now implements Timer interface.
func (t *timer) Now() time.Time {
	return time.Now()
}
