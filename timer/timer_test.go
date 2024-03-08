package timer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimer_Reset(t *testing.T) {
	tt := New()

	tt.Reset(1, 2, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	shouldReceive(t, tt, 1, 2, "no value in timer")

	tt.Reset(1, 2, time.Second)
	tt.Reset(2, 3, 0)
	shouldReceive(t, tt, 2, 3, "no value in timer after reset(0)")

	tt.Reset(1, 2, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	tt.Reset(1, 3, time.Millisecond*100)
	tt.Sleep(time.Millisecond * 200)
	shouldReceive(t, tt, 1, 3, "invalid value after reset")

	tt.Reset(3, 1, time.Millisecond*100)
	shouldNotReceive(t, tt, "value arrived too early")

	tt.Extend(time.Millisecond * 300)
	tt.Sleep(time.Millisecond * 200)
	shouldNotReceive(t, tt, "value arrived too early after extend")

	tt.Sleep(time.Millisecond * 300)
	shouldReceive(t, tt, 3, 1, "no value in timer after extend")

	tt.Reset(1, 1, time.Millisecond*100)
	tt.Stop()
	tt.Sleep(time.Millisecond * 200)
	shouldNotReceive(t, tt, "timer was not stopped")
}

func shouldReceive(t *testing.T, tt *Timer, height uint32, view byte, msg string) {
	select {
	case <-tt.C():
		gotHeight := tt.Height()
		gotView := tt.View()
		require.Equal(t, height, gotHeight)
		require.Equal(t, view, gotView)
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
