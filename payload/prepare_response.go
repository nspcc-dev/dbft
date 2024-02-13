package payload

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft/crypto"
)

// PrepareResponse represents dBFT PrepareResponse message.
type PrepareResponse[H crypto.Hash] interface {
	// PreparationHash returns the hash of PrepareRequest payload
	// for this epoch.
	PreparationHash() H
	// SetPreparationHash sets preparations hash.
	SetPreparationHash(h H)
}

type (
	prepareResponse struct {
		preparationHash crypto.Uint256
	}
	// prepareResponseAux is an auxiliary structure for prepareResponse encoding.
	prepareResponseAux struct {
		PreparationHash crypto.Uint256
	}
)

var _ PrepareResponse[crypto.Uint256] = (*prepareResponse)(nil)

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

// SetPreparationHash implements PrepareResponse interface.
func (p *prepareResponse) SetPreparationHash(h crypto.Uint256) {
	p.preparationHash = h
}
