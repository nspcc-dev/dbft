package crypto

import (
	"crypto/sha256"
	"encoding/hex"
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
	var (
		h1 = sha256.Sum256(data)
		h  Uint160
	)

	copy(h[:], h1[:Uint160Size])

	return h
}
