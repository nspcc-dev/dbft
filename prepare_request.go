package dbft

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest[H Hash, A Address] interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// SetTimestamp sets timestamp of this message.
	SetTimestamp(ts uint64)

	// Nonce is a random nonce.
	Nonce() uint64
	// SetNonce sets Nonce.
	SetNonce(nonce uint64)

	// TransactionHashes returns hashes of all transaction in a proposed block.
	TransactionHashes() []H
	// SetTransactionHashes sets transaction's hashes.
	SetTransactionHashes(hs []H)

	// NextConsensus returns hash which is based on which validators will
	// try to agree on a block in the current epoch.
	NextConsensus() A
	// SetNextConsensus sets next consensus field.
	SetNextConsensus(nc A)
}
