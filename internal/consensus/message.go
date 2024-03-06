package consensus

import (
	"bytes"
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	// Payload represents minimal payload containing all necessary fields.
	Payload struct {
		message

		version        uint32
		validatorIndex uint16
		prevHash       crypto.Uint256
		height         uint32

		hash *crypto.Uint256
	}

	// payloadAux is an auxiliary structure for Payload encoding.
	payloadAux struct {
		Version        uint32
		ValidatorIndex uint16
		PrevHash       crypto.Uint256
		Height         uint32

		Data []byte
	}
)

var _ dbft.ConsensusPayload[crypto.Uint256] = (*Payload)(nil)

// EncodeBinary implements Serializable interface.
func (p Payload) EncodeBinary(w *gob.Encoder) error {
	ww := bytes.Buffer{}
	enc := gob.NewEncoder(&ww)
	if err := p.message.EncodeBinary(enc); err != nil {
		return err
	}

	return w.Encode(&payloadAux{
		Version:        p.version,
		ValidatorIndex: p.validatorIndex,
		PrevHash:       p.prevHash,
		Height:         p.height,
		Data:           ww.Bytes(),
	})
}

// DecodeBinary implements Serializable interface.
func (p *Payload) DecodeBinary(r *gob.Decoder) error {
	aux := new(payloadAux)
	if err := r.Decode(aux); err != nil {
		return err
	}

	p.version = aux.Version
	p.prevHash = aux.PrevHash
	p.height = aux.Height
	p.validatorIndex = aux.ValidatorIndex

	rr := bytes.NewReader(aux.Data)
	dec := gob.NewDecoder(rr)
	return p.message.DecodeBinary(dec)
}

// MarshalUnsigned implements ConsensusPayload interface.
func (p Payload) MarshalUnsigned() []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	_ = p.EncodeBinary(enc)

	return buf.Bytes()
}

// UnmarshalUnsigned implements ConsensusPayload interface.
func (p *Payload) UnmarshalUnsigned(data []byte) error {
	r := bytes.NewReader(data)
	dec := gob.NewDecoder(r)
	return p.DecodeBinary(dec)
}

// Hash implements ConsensusPayload interface.
func (p *Payload) Hash() crypto.Uint256 {
	if p.hash != nil {
		return *p.hash
	}

	data := p.MarshalUnsigned()

	return crypto.Hash256(data)
}

// Version implements ConsensusPayload interface.
func (p Payload) Version() uint32 {
	return p.version
}

// ValidatorIndex implements ConsensusPayload interface.
func (p Payload) ValidatorIndex() uint16 {
	return p.validatorIndex
}

// SetValidatorIndex implements ConsensusPayload interface.
func (p *Payload) SetValidatorIndex(i uint16) {
	p.validatorIndex = i
}

// PrevHash implements ConsensusPayload interface.
func (p Payload) PrevHash() crypto.Uint256 {
	return p.prevHash
}

// Height implements ConsensusPayload interface.
func (p Payload) Height() uint32 {
	return p.height
}
