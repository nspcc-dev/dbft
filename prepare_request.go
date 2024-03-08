package dbft

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest[H Hash] interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// Nonce is a random nonce.
	Nonce() uint64
	// TransactionHashes returns hashes of all transaction in a proposed block.
	TransactionHashes() []H
}
