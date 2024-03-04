package crypto

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // SA1019: package golang.org/x/crypto/ripemd160 is deprecated
)

const (
	Uint256Size = 32
	Uint160Size = 20
)

type (
	Uint256 [Uint256Size]byte
	Uint160 [Uint160Size]byte
)

// String implements fmt.Stringer interface.
func (h Uint256) String() string {
	return hex.EncodeToString(h[:])
}

// String implements fmt.Stringer interface.
func (h Uint160) String() string {
	return hex.EncodeToString(h[:])
}

// Hash256 returns double sha-256 of data.
func Hash256(data []byte) Uint256 {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])

	return h2
}

// Hash160 returns ripemd160 from sha256 of data.
func Hash160(data []byte) Uint160 {
	h1 := sha256.Sum256(data)
	rp := ripemd160.New()
	_, _ = rp.Write(h1[:])

	var h Uint160
	copy(h[:], rp.Sum(nil))

	return h
}
