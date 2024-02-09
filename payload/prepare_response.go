package payload

import (
	"encoding/gob"

	"github.com/nspcc-dev/neo-go/pkg/util"
)

// PrepareResponse represents dBFT PrepareResponse message.
type PrepareResponse interface {
	// PreparationHash returns the hash of PrepareRequest payload
	// for this epoch.
	PreparationHash() util.Uint256
	// SetPreparationHash sets preparations hash.
	SetPreparationHash(h util.Uint256)
}

type (
	prepareResponse struct {
		preparationHash util.Uint256
	}
	// prepareResponseAux is an auxiliary structure for prepareResponse encoding.
	prepareResponseAux struct {
		PreparationHash util.Uint256
	}
)

var _ PrepareResponse = (*prepareResponse)(nil)

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
func (p *prepareResponse) PreparationHash() util.Uint256 {
	return p.preparationHash
}

// SetPreparationHash implements PrepareResponse interface.
func (p *prepareResponse) SetPreparationHash(h util.Uint256) {
	p.preparationHash = h
}
