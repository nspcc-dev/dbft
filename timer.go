package dbft

import (
	"time"
)

// Timer is an interface which implements all time-related
// functions. It can be mocked for testing.
type Timer interface {
	// Now returns current time.
	Now() time.Time
	// Reset resets timer to the specified block height and view.
	Reset(hv HV, d time.Duration)
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

// HV is an abstraction for pair of a Height and a View.
type HV interface {
	Height() uint32
	View() byte
}
