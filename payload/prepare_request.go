package payload

import (
	"encoding/gob"

	"github.com/nspcc-dev/neo-go/pkg/util"
)

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// SetTimestamp sets timestamp of this message.
	SetTimestamp(ts uint64)

	// Nonce is a random nonce.
	Nonce() uint64
	// SetNonce sets Nonce.
	SetNonce(nonce uint64)

	// TransactionHashes returns hashes of all transaction in a proposed block.
	TransactionHashes() []util.Uint256
	// SetTransactionHashes sets transaction's hashes.
	SetTransactionHashes(hs []util.Uint256)

	// NextConsensus returns hash which is based on which validators will
	// try to agree on a block in the current epoch.
	NextConsensus() util.Uint160
	// SetNextConsensus sets next consensus field.
	SetNextConsensus(nc util.Uint160)
}

type (
	prepareRequest struct {
		transactionHashes []util.Uint256
		nonce             uint64
		timestamp         uint32
		nextConsensus     util.Uint160
	}
	// prepareRequestAux is an auxiliary structure for prepareRequest encoding.
	prepareRequestAux struct {
		TransactionHashes []util.Uint256
		Nonce             uint64
		Timestamp         uint32
		NextConsensus     util.Uint160
	}
)

var _ PrepareRequest = (*prepareRequest)(nil)

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
func (p prepareRequest) TransactionHashes() []util.Uint256 {
	return p.transactionHashes
}

// SetTransactionHashes implements PrepareRequest interface.
func (p *prepareRequest) SetTransactionHashes(hs []util.Uint256) {
	p.transactionHashes = hs
}

// NextConsensus implements PrepareRequest interface.
func (p prepareRequest) NextConsensus() util.Uint160 {
	return p.nextConsensus
}

// SetNextConsensus implements PrepareRequest interface.
func (p *prepareRequest) SetNextConsensus(nc util.Uint160) {
	p.nextConsensus = nc
}
