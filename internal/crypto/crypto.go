package crypto

import (
	"io"

	"github.com/nspcc-dev/dbft"
)

type suiteType byte

const (
	// SuiteECDSA is a ECDSA suite over P-256 curve
	// with 64-byte uncompressed signatures.
	SuiteECDSA suiteType = 1 + iota
)

const defaultSuite = SuiteECDSA

// Generate generates new key pair using r
// as a source of entropy.
func Generate(r io.Reader) (dbft.PrivateKey, dbft.PublicKey) {
	return GenerateWith(defaultSuite, r)
}

// GenerateWith generates new key pair for suite t
// using r as a source of entropy.
func GenerateWith(t suiteType, r io.Reader) (dbft.PrivateKey, dbft.PublicKey) {
	if t == SuiteECDSA {
		return generateECDSA(r)
	}

	return nil, nil
}
