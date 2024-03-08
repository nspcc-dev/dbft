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
	Reset(height uint32, view byte, d time.Duration)
	// Sleep stops execution for duration d.
	Sleep(d time.Duration)
	// Extend extends current timer with duration d.
	Extend(d time.Duration)
	// Stop stops timer.
	Stop()
	// Height returns current height set for the timer.
	Height() uint32
	// View returns current view set for the timer.
	View() byte
	// C returns channel for timer events.
	C() <-chan time.Time
}
