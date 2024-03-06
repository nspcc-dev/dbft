package consensus

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
)

type (
	commit struct {
		signature [signatureSize]byte
	}
	// commitAux is an auxiliary structure for commit encoding.
	commitAux struct {
		Signature [signatureSize]byte
	}
)

const signatureSize = 64

var _ dbft.Commit = (*commit)(nil)

// EncodeBinary implements Serializable interface.
func (c commit) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(commitAux{
		Signature: c.signature,
	})
}

// DecodeBinary implements Serializable interface.
func (c *commit) DecodeBinary(r *gob.Decoder) error {
	aux := new(commitAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	c.signature = aux.Signature
	return nil
}

// Signature implements Commit interface.
func (c commit) Signature() []byte {
	return c.signature[:]
}
