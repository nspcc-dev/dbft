package consensus

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	prepareRequest struct {
		transactionHashes []crypto.Uint256
		nonce             uint64
		timestamp         uint32
	}
	// prepareRequestAux is an auxiliary structure for prepareRequest encoding.
	prepareRequestAux struct {
		TransactionHashes []crypto.Uint256
		Nonce             uint64
		Timestamp         uint32
	}
)

var _ dbft.PrepareRequest[crypto.Uint256] = (*prepareRequest)(nil)

// EncodeBinary implements Serializable interface.
func (p prepareRequest) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(&prepareRequestAux{
		TransactionHashes: p.transactionHashes,
		Nonce:             p.nonce,
		Timestamp:         p.timestamp,
	})
}

// DecodeBinary implements Serializable interface.
func (p *prepareRequest) DecodeBinary(r *gob.Decoder) error {
	aux := new(prepareRequestAux)
	if err := r.Decode(aux); err != nil {
		return err
	}

	p.timestamp = aux.Timestamp
	p.nonce = aux.Nonce
	p.transactionHashes = aux.TransactionHashes
	return nil
}

// Timestamp implements PrepareRequest interface.
func (p prepareRequest) Timestamp() uint64 {
	return secToNanoSec(p.timestamp)
}

// Nonce implements PrepareRequest interface.
func (p prepareRequest) Nonce() uint64 {
	return p.nonce
}

// TransactionHashes implements PrepareRequest interface.
func (p prepareRequest) TransactionHashes() []crypto.Uint256 {
	return p.transactionHashes
}
