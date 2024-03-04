package payload

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
		nextConsensus     crypto.Uint160
	}
	// prepareRequestAux is an auxiliary structure for prepareRequest encoding.
	prepareRequestAux struct {
		TransactionHashes []crypto.Uint256
		Nonce             uint64
		Timestamp         uint32
		NextConsensus     crypto.Uint160
	}
)

var _ dbft.PrepareRequest[crypto.Uint256, crypto.Uint160] = (*prepareRequest)(nil)

// EncodeBinary implements Serializable interface.
func (p prepareRequest) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(&prepareRequestAux{
		TransactionHashes: p.transactionHashes,
		Nonce:             p.nonce,
		Timestamp:         p.timestamp,
		NextConsensus:     p.nextConsensus,
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
	p.nextConsensus = aux.NextConsensus
	p.transactionHashes = aux.TransactionHashes
	return nil
}

// Timestamp implements PrepareRequest interface.
func (p prepareRequest) Timestamp() uint64 {
	return secToNanoSec(p.timestamp)
}

// SetTimestamp implements PrepareRequest interface.
func (p *prepareRequest) SetTimestamp(ts uint64) {
	p.timestamp = nanoSecToSec(ts)
}

// Nonce implements PrepareRequest interface.
func (p prepareRequest) Nonce() uint64 {
	return p.nonce
}

// SetNonce implements PrepareRequest interface.
func (p *prepareRequest) SetNonce(nonce uint64) {
	p.nonce = nonce
}

// TransactionHashes implements PrepareRequest interface.
func (p prepareRequest) TransactionHashes() []crypto.Uint256 {
	return p.transactionHashes
}

// SetTransactionHashes implements PrepareRequest interface.
func (p *prepareRequest) SetTransactionHashes(hs []crypto.Uint256) {
	p.transactionHashes = hs
}

// NextConsensus implements PrepareRequest interface.
func (p prepareRequest) NextConsensus() crypto.Uint160 {
	return p.nextConsensus
}

// SetNextConsensus implements PrepareRequest interface.
func (p *prepareRequest) SetNextConsensus(nc crypto.Uint160) {
	p.nextConsensus = nc
}
