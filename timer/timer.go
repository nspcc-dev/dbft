package timer

import (
	"sync"
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

	// HV is a pair of a Height and a View.
	HV struct {
		Height uint32
		View   byte
	}

	timer struct {
		*sync.RWMutex
		s        HV
		start    time.Time
		duration time.Duration
		t        *time.Timer
		ch       chan HV
		stop     chan struct{}
	}
)

var _ Timer = (*timer)(nil)

// New returns default Timer implementation.
func New() Timer {
	return &timer{ch: make(chan HV, 1), RWMutex: new(sync.RWMutex)}
}

// C implements Timer interface.
func (t *timer) C() <-chan HV { return (<-chan HV)(t.ch) }

// Reset implements Timer interface.
func (t *timer) Reset(s HV, d time.Duration) {
	t.reset(s, time.Now(), d)
}

// Stop implements Timer interface.
func (t *timer) Stop() {
	t.Lock()
	if t.stop != nil {
		close(t.stop)
	}
	t.Unlock()

	t.timerStop()
}

// Sleep implements Timer interface.
func (t *timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (t *timer) isStopped() <-chan struct{} {
	t.RLock()
	defer t.RUnlock()

	if t.stop == nil {
		return nil
	}

	return t.stop
}

func (t *timer) timerReset(s HV, start time.Time, d time.Duration) {
	elapsed := time.Since(start)
	if d <= elapsed {
		t.s = s
		return
	}

	t.Stop()

	t.Lock()
	t.start = start
	t.duration = d
	t.s = s
	t.stop = make(chan struct{})
	t.t = time.NewTimer(d - elapsed)
	t.Unlock()
}

func (t *timer) timerStop() {
	t.Lock()
	if t.t != nil {
		if !t.t.Stop() {
			//<-t.t.C
		}
	}
	t.Unlock()
}

func (t *timer) timerChannel() <-chan time.Time {
	t.RLock()
	defer t.RUnlock()

	if t.t == nil {
		return nil
	}

	return t.t.C
}

func (t *timer) reset(s HV, start time.Time, d time.Duration) {
	if d == 0 {
		t.timerStop()

		if t.t != nil {
			t.t.Stop()
		}

		select {
		case <-t.ch:
		default:
		}

		t.start = start
		t.s = s
		t.ch <- s

		return
	}

	t.timerReset(s, start, d)

	go func() {
		select {
		case _, ok := <-t.timerChannel():
			if ok {
				t.timerStop()
				t.ch <- t.s
			}
		case <-t.isStopped():
		}
	}()
}

// Extend implements Timer interface.
func (t *timer) Extend(d time.Duration) {
	t.reset(t.s, t.start, d*t.duration)
}

// Now implements Timer interface.
func (t *timer) Now() time.Time {
	return time.Now()
}
