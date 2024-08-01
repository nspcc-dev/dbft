package consensus

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
)

type (
	// amevCommit implements dbft.Commit.
	amevCommit struct {
		data [dataSize]byte
	}
	// amevCommitAux is an auxiliary structure for amevCommit encoding.
	amevCommitAux struct {
		Data [dataSize]byte
	}
)

const dataSize = 64

var _ dbft.Commit = (*amevCommit)(nil)

// EncodeBinary implements Serializable interface.
func (c amevCommit) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(amevCommitAux{
		Data: c.data,
	})
}

// DecodeBinary implements Serializable interface.
func (c *amevCommit) DecodeBinary(r *gob.Decoder) error {
	aux := new(amevCommitAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	c.data = aux.Data
	return nil
}

// Signature implements Commit interface.
func (c amevCommit) Signature() []byte {
	return c.data[:]
}
