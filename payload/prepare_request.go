package payload

import (
	"github.com/nspcc-dev/neo-go/pkg/io"
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

type prepareRequest struct {
	transactionHashes []util.Uint256
	nonce             uint64
	timestamp         uint32
	nextConsensus     util.Uint160
}

var _ PrepareRequest = (*prepareRequest)(nil)

// EncodeBinary implements io.Serializable interface.
func (p prepareRequest) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(p.timestamp)
	w.WriteU64LE(p.nonce)
	w.WriteBytes(p.nextConsensus[:])
	w.WriteArray(p.transactionHashes)
}

// DecodeBinary implements io.Serializable interface.
func (p *prepareRequest) DecodeBinary(r *io.BinReader) {
	p.timestamp = r.ReadU32LE()
	p.nonce = r.ReadU64LE()
	r.ReadBytes(p.nextConsensus[:])
	r.ReadArray(&p.transactionHashes)
}

// Timestamp implements PrepareRequest interface.
func (p prepareRequest) Timestamp() uint64 {
	return uint64(p.timestamp)
}

// SetTimestamp implements PrepareRequest interface.
func (p *prepareRequest) SetTimestamp(ts uint64) {
	p.timestamp = uint32(ts)
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
