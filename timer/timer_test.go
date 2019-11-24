package timer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimer_Reset(t *testing.T) {
	tt := New()

	tt.Reset(HV{Height: 1, View: 2}, time.Millisecond*100)
	time.Sleep(time.Millisecond * 200)
	select {
	case hv := <-tt.C():
		require.Equal(t, hv, HV{Height: 1, View: 2})
	default:
		require.Fail(t, "no value in timer")
	}

	tt.Reset(HV{Height: 1, View: 2}, time.Second)
	tt.Reset(HV{Height: 2, View: 3}, 0)
	select {
	case hv := <-tt.C():
		require.Equal(t, hv, HV{Height: 2, View: 3})
	default:
		require.Fail(t, "no value in timer after reset(0)")
	}

	tt.Reset(HV{Height: 3, View: 1}, time.Millisecond*100)
	select {
	case <-tt.C():
		require.Fail(t, "value arrived to early")
	default:
	}

	time.Sleep(time.Millisecond * 200)
	select {
	case hv := <-tt.C():
		require.Equal(t, hv, HV{Height: 3, View: 1})
	default:
		require.Fail(t, "no value in timer after reset")
	}
}
