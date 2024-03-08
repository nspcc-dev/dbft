package consensus

import (
	"encoding/gob"
)

type (
	changeViewCompact struct {
		ValidatorIndex     uint16
		OriginalViewNumber byte
		Timestamp          uint32
	}

	commitCompact struct {
		ViewNumber     byte
		ValidatorIndex uint16
		Signature      [signatureSize]byte
	}

	preparationCompact struct {
		ValidatorIndex uint16
	}
)

// EncodeBinary implements Serializable interface.
func (p changeViewCompact) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(p)
}

// DecodeBinary implements Serializable interface.
func (p *changeViewCompact) DecodeBinary(r *gob.Decoder) error {
	return r.Decode(p)
}

// EncodeBinary implements Serializable interface.
func (p commitCompact) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(p)
}

// DecodeBinary implements Serializable interface.
func (p *commitCompact) DecodeBinary(r *gob.Decoder) error {
	return r.Decode(p)
}

// EncodeBinary implements Serializable interface.
func (p preparationCompact) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(p)
}

// DecodeBinary implements Serializable interface.
func (p *preparationCompact) DecodeBinary(r *gob.Decoder) error {
	return r.Decode(p)
}
