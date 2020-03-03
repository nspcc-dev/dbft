package payload

import "github.com/nspcc-dev/neo-go/pkg/io"

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
	w.WriteU16LE(p.validatorIndex)
	w.WriteB(p.originalViewNumber)
	w.WriteU32LE(p.timestamp)
}

// DecodeBinary implements io.Serializable interface.
func (p *changeViewCompact) DecodeBinary(r *io.BinReader) {
	p.validatorIndex = r.ReadU16LE()
	p.originalViewNumber = r.ReadB()
	p.timestamp = r.ReadU32LE()
}

// EncodeBinary implements io.Serializable interface.
func (p commitCompact) EncodeBinary(w *io.BinWriter) {
	w.WriteB(p.viewNumber)
	w.WriteU16LE(p.validatorIndex)
	w.WriteBytes(p.signature[:])
}

// DecodeBinary implements io.Serializable interface.
func (p *commitCompact) DecodeBinary(r *io.BinReader) {
	p.viewNumber = r.ReadB()
	p.validatorIndex = r.ReadU16LE()
	r.ReadBytes(p.signature[:])
}

// EncodeBinary implements io.Serializable interface.
func (p preparationCompact) EncodeBinary(w *io.BinWriter) {
	w.WriteU16LE(p.validatorIndex)
}

// DecodeBinary implements io.Serializable interface.
func (p *preparationCompact) DecodeBinary(r *io.BinReader) {
	p.validatorIndex = r.ReadU16LE()
}
