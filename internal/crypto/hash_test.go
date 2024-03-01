package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

var hash256tc = []struct {
	data []byte
	hash Uint256
}{
	{[]byte{}, parse256("5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456")},
	{[]byte{0, 1, 2, 3}, parse256("f7a355c00c89a08c80636bed35556a210b51786f6803a494f28fc5ba05959fc2")},
}

var hash160tc = []struct {
	data []byte
	hash Uint160
}{
	{[]byte{}, parse160("b472a266d0bd89c13706a4132ccfb16f7c3b9fcb")},
	{[]byte{0, 1, 2, 3}, parse160("3c3fa3d4adcaf8f52d5b1843975e122548269937")},
}

func TestHash256(t *testing.T) {
	for _, tc := range hash256tc {
		require.Equal(t, tc.hash, Hash256(tc.data))
	}
}

func TestHash160(t *testing.T) {
	for _, tc := range hash160tc {
		require.Equal(t, tc.hash, Hash160(tc.data))
	}
}

func parse256(s string) (h Uint256) {
	parseHex(h[:], s)
	return
}

func parse160(s string) (h Uint160) {
	parseHex(h[:], s)
	return
}

func parseHex(b []byte, s string) {
	buf, err := hex.DecodeString(s)
	if err != nil || len(buf) != len(b) {
		panic("invalid test data")
	}

	copy(b, buf)
}
