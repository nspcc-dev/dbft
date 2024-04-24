package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"math/big"

	"github.com/nspcc-dev/dbft"
)

type (
	// ECDSAPub is a wrapper over *ecsda.PublicKey.
	ECDSAPub struct {
		*ecdsa.PublicKey
	}

	// ECDSAPriv is a wrapper over *ecdsa.PrivateKey.
	ECDSAPriv struct {
		*ecdsa.PrivateKey
	}
)

func generateECDSA(r io.Reader) (dbft.PrivateKey, dbft.PublicKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), r)
	if err != nil {
		return nil, nil
	}

	return NewECDSAPrivateKey(key), NewECDSAPublicKey(&key.PublicKey)
}

// NewECDSAPublicKey returns new PublicKey from *ecdsa.PublicKey.
func NewECDSAPublicKey(pub *ecdsa.PublicKey) dbft.PublicKey {
	return &ECDSAPub{
		PublicKey: pub,
	}
}

// NewECDSAPrivateKey returns new PublicKey from *ecdsa.PrivateKey.
func NewECDSAPrivateKey(key *ecdsa.PrivateKey) dbft.PrivateKey {
	return &ECDSAPriv{
		PrivateKey: key,
	}
}

// Sign signs message using P-256 curve.
func (e ECDSAPriv) Sign(msg []byte) ([]byte, error) {
	h := sha256.Sum256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, e.PrivateKey, h[:])
	if err != nil {
		return nil, err
	}

	sig := make([]byte, 32*2)
	_ = r.FillBytes(sig[:32])
	_ = s.FillBytes(sig[32:])

	return sig, nil
}

// Equals implements dbft.PublicKey interface.
func (e *ECDSAPub) Equals(other dbft.PublicKey) bool {
	return e.Equal(other.(*ECDSAPub).PublicKey)
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (e ECDSAPub) MarshalBinary() ([]byte, error) {
	return elliptic.MarshalCompressed(e.PublicKey.Curve, e.PublicKey.X, e.PublicKey.Y), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (e *ECDSAPub) UnmarshalBinary(data []byte) error {
	e.PublicKey = new(ecdsa.PublicKey)
	e.PublicKey.Curve = elliptic.P256()
	e.PublicKey.X, e.PublicKey.Y = elliptic.UnmarshalCompressed(e.PublicKey.Curve, data)
	if e.PublicKey.X == nil {
		return errors.New("can't unmarshal ECDSA public key")
	}

	return nil
}

// Verify verifies signature using P-256 curve.
func (e ECDSAPub) Verify(msg, sig []byte) error {
	h := sha256.Sum256(msg)
	rBytes := new(big.Int).SetBytes(sig[0:32])
	sBytes := new(big.Int).SetBytes(sig[32:64])
	res := ecdsa.Verify(e.PublicKey, h[:], rBytes, sBytes)
	if !res {
		return errors.New("bad signature")
	}
	return nil
}
