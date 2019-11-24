package payload

import (
	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/CityOfZion/neo-go/pkg/util"
)

// PrepareResponse represents dBFT PrepareResponse message.
type PrepareResponse interface {
	// PreparationHash returns the hash of PrepareRequest payload
	// for this epoch.
	PreparationHash() util.Uint256
	// SetPreparationHash sets preparations hash.
	SetPreparationHash(h util.Uint256)
}

type prepareResponse struct {
	preparationHash util.Uint256
}

var _ PrepareResponse = (*prepareResponse)(nil)

// EncodeBinary implements io.Serializable interface.
func (p prepareResponse) EncodeBinary(w *io.BinWriter) {
	p.preparationHash.EncodeBinary(w)
}

// DecodeBinary implements io.Serializable interface.
func (p *prepareResponse) DecodeBinary(r *io.BinReader) {
	p.preparationHash.DecodeBinary(r)
}

// PreparationHash implements PrepareResponse interface.
func (p *prepareResponse) PreparationHash() util.Uint256 {
	return p.preparationHash
}

// SetPreparationHash implements PrepareResponse interface.
func (p *prepareResponse) SetPreparationHash(h util.Uint256) {
	p.preparationHash = h
}
