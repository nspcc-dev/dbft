package dbft

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest[H Hash] interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// Nonce is a random nonce.
	Nonce() uint64
	// Transactions returns the list of all transactions in a proposed block
	// with possible gaps in place of missing transactions and the map of
	// missing transaction hashes to their indexes in the proposal list.
	Transactions() ([]Transaction[H], map[H]int)
}
