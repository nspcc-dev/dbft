package payload

import (
	"bytes"
	"encoding/gob"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

type (
	// ConsensusPayload is a generic payload type which is exchanged
	// between the nodes.
	ConsensusPayload interface {
		consensusMessage

		// ValidatorIndex returns index of validator from which
		// payload was originated from.
		ValidatorIndex() uint16

		// SetValidator index sets validator index.
		SetValidatorIndex(i uint16)

		Height() uint32
		SetHeight(h uint32)

		// Hash returns 32-byte checksum of the payload.
		Hash() util.Uint256
	}

	// Payload represents minimal payload containing all necessary fields.
	Payload struct {
		message

		version        uint32
		validatorIndex uint16
		prevHash       util.Uint256
		height         uint32

		hash *util.Uint256
	}

	// payloadAux is an auxiliary structure for Payload encoding.
	payloadAux struct {
		Version        uint32
		ValidatorIndex uint16
		PrevHash       util.Uint256
		Height         uint32

		Data []byte
	}
)

var _ ConsensusPayload = (*Payload)(nil)

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
func (p *Payload) Hash() util.Uint256 {
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

// SetVersion implements ConsensusPayload interface.
func (p *Payload) SetVersion(v uint32) {
	p.version = v
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
func (p Payload) PrevHash() util.Uint256 {
	return p.prevHash
}

// SetPrevHash implements ConsensusPayload interface.
func (p *Payload) SetPrevHash(h util.Uint256) {
	p.prevHash = h
}

// Height implements ConsensusPayload interface.
func (p Payload) Height() uint32 {
	return p.height
}

// SetHeight implements ConsensusPayload interface.
func (p *Payload) SetHeight(h uint32) {
	p.height = h
}
