package dbft

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest[H Hash] interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// Nonce is a random nonce.
	Nonce() uint64
	// TransactionHashes returns hashes of all transaction in a proposed block.
	// It's used when PrepareRequestExtensionEnabled is false.
	TransactionHashes() []H
	// Transactions returns full transaction list attached to this PrepareRequest.
	// It's used when PrepareRequestExtensionEnabled is true.
	Transactions() []Transaction[H]
}
