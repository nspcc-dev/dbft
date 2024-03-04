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
	{[]byte{}, Uint160{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4}},
	{[]byte{0, 1, 2, 3}, Uint160{0x5, 0x4e, 0xde, 0xc1, 0xd0, 0x21, 0x1f, 0x62, 0x4f, 0xed, 0xc, 0xbc, 0xa9, 0xd4, 0xf9, 0x40, 0xb, 0xe, 0x49, 0x1c}},
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

func parseHex(b []byte, s string) {
	buf, err := hex.DecodeString(s)
	if err != nil || len(buf) != len(b) {
		panic("invalid test data")
	}

	copy(b, buf)
}
