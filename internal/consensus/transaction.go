package consensus

import (
	"encoding/binary"
	"errors"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

// =============================
// Small transaction.
// =============================

type Tx64 uint64

var _ dbft.Transaction[crypto.Uint256] = (*Tx64)(nil)

func (t *Tx64) Hash() (h crypto.Uint256) {
	binary.LittleEndian.PutUint64(h[:], uint64(*t))
	return
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (t *Tx64) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(*t))

	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnarshaler interface.
func (t *Tx64) UnmarshalBinary(data []byte) error {
	if len(data) != 8 {
		return errors.New("length must equal 8 bytes")
	}

	*t = Tx64(binary.LittleEndian.Uint64(data))

	return nil
}
