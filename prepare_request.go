package dbft

// PrepareRequest represents dBFT PrepareRequest message.
type PrepareRequest[H Hash, Tx Transaction[H]] interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// Nonce is a random nonce.
	Nonce() uint64
	// Transactions returns the list of all transactions in a proposed block
	// with possible gaps in place of missing transactions and the map of
	// missing transaction hashes to their indexes in the proposal list.
	Transactions() ([]Tx, map[H]int)
}
