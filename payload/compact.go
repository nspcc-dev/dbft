package payload

import "github.com/CityOfZion/neo-go/pkg/io"

type (
	changeViewCompact struct {
		validatorIndex     uint16
		originalViewNumber byte
		timestamp          uint32
	}

	commitCompact struct {
		viewNumber     byte
		validatorIndex uint16
		signature      [signatureSize]byte
	}

	preparationCompact struct {
		validatorIndex uint16
	}
)

// EncodeBinary implements io.Serializable interface.
func (p changeViewCompact) EncodeBinary(w *io.BinWriter) {
	w.WriteLE(p.validatorIndex)
	w.WriteLE(p.originalViewNumber)
	w.WriteLE(p.timestamp)
}

// DecodeBinary implements io.Serializable interface.
func (p *changeViewCompact) DecodeBinary(r *io.BinReader) {
	r.ReadLE(&p.validatorIndex)
	r.ReadLE(&p.originalViewNumber)
	r.ReadLE(&p.timestamp)
}

// EncodeBinary implements io.Serializable interface.
func (p commitCompact) EncodeBinary(w *io.BinWriter) {
	w.WriteLE(p.viewNumber)
	w.WriteLE(p.validatorIndex)
	w.WriteBE(p.signature)
}

// DecodeBinary implements io.Serializable interface.
func (p *commitCompact) DecodeBinary(r *io.BinReader) {
	r.ReadLE(&p.viewNumber)
	r.ReadLE(&p.validatorIndex)
	r.ReadBE(p.signature[:])
}

// EncodeBinary implements io.Serializable interface.
func (p preparationCompact) EncodeBinary(w *io.BinWriter) {
	w.WriteLE(p.validatorIndex)
}

// DecodeBinary implements io.Serializable interface.
func (p *preparationCompact) DecodeBinary(r *io.BinReader) {
	r.ReadLE(&p.validatorIndex)
}
