package consensus

import (
	"encoding/binary"
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
)

type (
	// preCommit implements dbft.PreCommit.
	preCommit struct {
		magic uint32 // some magic data CN have to exchange to properly construct final amevBlock.
	}
	// preCommitAux is an auxiliary structure for preCommit encoding.
	preCommitAux struct {
		Magic uint32
	}
)

var _ dbft.PreCommit = (*preCommit)(nil)

// EncodeBinary implements Serializable interface.
func (c preCommit) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(preCommitAux{
		Magic: c.magic,
	})
}

// DecodeBinary implements Serializable interface.
func (c *preCommit) DecodeBinary(r *gob.Decoder) error {
	aux := new(preCommitAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	c.magic = aux.Magic
	return nil
}

// Data implements PreCommit interface.
func (c preCommit) Data() []byte {
	res := make([]byte, 4)
	binary.BigEndian.PutUint32(res, c.magic)
	return res
}
