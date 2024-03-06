package consensus

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	prepareResponse struct {
		preparationHash crypto.Uint256
	}
	// prepareResponseAux is an auxiliary structure for prepareResponse encoding.
	prepareResponseAux struct {
		PreparationHash crypto.Uint256
	}
)

var _ dbft.PrepareResponse[crypto.Uint256] = (*prepareResponse)(nil)

// EncodeBinary implements Serializable interface.
func (p prepareResponse) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(prepareResponseAux{
		PreparationHash: p.preparationHash,
	})
}

// DecodeBinary implements Serializable interface.
func (p *prepareResponse) DecodeBinary(r *gob.Decoder) error {
	aux := new(prepareResponseAux)
	if err := r.Decode(aux); err != nil {
		return err
	}

	p.preparationHash = aux.PreparationHash
	return nil
}

// PreparationHash implements PrepareResponse interface.
func (p *prepareResponse) PreparationHash() crypto.Uint256 {
	return p.preparationHash
}
