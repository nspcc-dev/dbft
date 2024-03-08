package timer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimer_Reset(t *testing.T) {
	tt := New()

	tt.Reset(HV{H: 1, V: 2}, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	shouldReceive(t, tt, HV{H: 1, V: 2}, "no value in timer")

	tt.Reset(HV{H: 1, V: 2}, time.Second)
	tt.Reset(HV{H: 2, V: 3}, 0)
	shouldReceive(t, tt, HV{H: 2, V: 3}, "no value in timer after reset(0)")

	tt.Reset(HV{H: 1, V: 2}, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	tt.Reset(HV{H: 1, V: 3}, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	shouldReceive(t, tt, HV{H: 1, V: 3}, "invalid value after reset")

	tt.Reset(HV{H: 3, V: 1}, time.Millisecond*100)
	shouldNotReceive(t, tt, "value arrived too early")

	tt.Extend(time.Millisecond * 300)
	tt.Sleep(time.Millisecond * 200)
	shouldNotReceive(t, tt, "value arrived too early after extend")

	tt.Sleep(time.Millisecond * 300)
	shouldReceive(t, tt, HV{H: 3, V: 1}, "no value in timer after extend")

	tt.Reset(HV{1, 1}, time.Millisecond*100)
	tt.Stop()
	tt.Sleep(time.Millisecond * 200)
	shouldNotReceive(t, tt, "timer was not stopped")
}

func shouldReceive(t *testing.T, tt *Timer, hv HV, msg string) {
	select {
	case <-tt.C():
		got := tt.HV()
		require.Equal(t, hv, got)
	default:
		require.Fail(t, msg)
	}
}

func shouldNotReceive(t *testing.T, tt *Timer, msg string) {
	select {
	case <-tt.C():
		require.Fail(t, msg)
	default:
	}
}
