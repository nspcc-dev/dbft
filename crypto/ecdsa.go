package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"io"

	crypto "github.com/nspcc-dev/neofs-crypto"
)

type (
	// ECDSAPub is a wrapper over *ecsda.PublicKey
	ECDSAPub struct {
		*ecdsa.PublicKey
	}

	// ECDSAPriv is a wrapper over *ecdsa.PrivateKey
	ECDSAPriv struct {
		*ecdsa.PrivateKey
	}
)

func generateECDSA(r io.Reader) (PrivateKey, PublicKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), r)
	if err != nil {
		return nil, nil
	}

	return NewECDSAPrivateKey(key), NewECDSAPublicKey(&key.PublicKey)
}

// NewECDSAPublicKey returns new PublicKey from *ecdsa.PublicKey.
func NewECDSAPublicKey(pub *ecdsa.PublicKey) PublicKey {
	return ECDSAPub{
		PublicKey: pub,
	}
}

// NewECDSAPrivateKey returns new PublicKey from *ecdsa.PrivateKey.
func NewECDSAPrivateKey(key *ecdsa.PrivateKey) PrivateKey {
	return ECDSAPriv{
		PrivateKey: key,
	}
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (e ECDSAPriv) MarshalBinary() (data []byte, err error) {
	return
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (e ECDSAPriv) UnmarshalBinary(data []byte) (err error) {
	e.PrivateKey, err = crypto.UnmarshalPrivateKey(data)
	return err
}

// Sign signs message using P-256 curve.
func (e ECDSAPriv) Sign(msg []byte) (sig []byte, err error) {
	sig, err = crypto.Sign(e.PrivateKey, msg)
	if err != nil {
		return nil, err
	}

	// we chomp first 0x04 (uncompressed) byte
	return sig[1:], err
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (e ECDSAPub) MarshalBinary() ([]byte, error) {
	return crypto.MarshalPublicKey(e.PublicKey), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (e ECDSAPub) UnmarshalBinary(data []byte) error {
	e.PublicKey = crypto.UnmarshalPublicKey(data)
	if e.PublicKey == nil {
		return errors.New("can't unmarshal ECDSA public key")
	}

	return nil
}

// Verify verifies signature using P-256 curve.
func (e ECDSAPub) Verify(msg, sig []byte) error {
	return crypto.Verify(e.PublicKey, msg, append([]byte{0x04}, sig...))
}
