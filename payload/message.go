package payload

import (
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

type (
	// ConsensusPayload is a generic payload type which is exchanged
	// between the nodes.
	ConsensusPayload interface {
		consensusMessage

		// MarshalUnsigned marshals payload into a byte array.
		// It MUST be stable and contain no signatures and other
		// fields which can be changed.
		MarshalUnsigned() []byte

		// UnmarshalUnsigned unmarshals payload from a byte array.
		UnmarshalUnsigned([]byte) error

		Version() uint32
		SetVersion(v uint32)

		// ValidatorIndex returns index of validator from which
		// payload was originated from.
		ValidatorIndex() uint16

		// SetValidator index sets validator index.
		SetValidatorIndex(i uint16)

		PrevHash() util.Uint256
		SetPrevHash(h util.Uint256)

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
)

var _ ConsensusPayload = (*Payload)(nil)

// EncodeBinary implements io.Serializable interface.
func (p Payload) EncodeBinary(w *io.BinWriter) {
	ww := io.NewBufBinWriter()
	p.message.EncodeBinary(ww.BinWriter)
	data := ww.Bytes()

	w.WriteU32LE(p.version)
	w.WriteBytes(p.prevHash[:])
	w.WriteU32LE(p.height)
	w.WriteU16LE(p.validatorIndex)
	w.WriteVarBytes(data)
}

// DecodeBinary implements io.Serializable interface.
func (p *Payload) DecodeBinary(r *io.BinReader) {
	p.version = r.ReadU32LE()
	p.prevHash.DecodeBinary(r)
	p.height = r.ReadU32LE()
	p.validatorIndex = r.ReadU16LE()

	data := r.ReadVarBytes()
	rr := io.NewBinReaderFromBuf(data)
	p.message.DecodeBinary(rr)
}

// MarshalUnsigned implements ConsensusPayload interface.
func (p Payload) MarshalUnsigned() []byte {
	w := io.NewBufBinWriter()
	p.EncodeBinary(w.BinWriter)

	return w.Bytes()
}

// UnmarshalUnsigned implements ConsensusPayload interface.
func (p *Payload) UnmarshalUnsigned(data []byte) error {
	r := io.NewBinReaderFromBuf(data)
	p.DecodeBinary(r)

	return r.Err
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
