package crypto

import (
	"crypto/sha256"

	"github.com/nspcc-dev/neo-go/pkg/util"
	"golang.org/x/crypto/ripemd160"
)

// Hash256 returns double sha-256 of data.
func Hash256(data []byte) util.Uint256 {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])

	return h2
}

// Hash160 returns ripemd160 from sha256 of data.
func Hash160(data []byte) (h util.Uint160) {
	h1 := sha256.Sum256(data)
	rp := ripemd160.New()
	_, _ = rp.Write(h1[:])
	copy(h[:], rp.Sum(nil))

	return
}
