package dbft

// Transaction is a generic transaction interface.
type Transaction[H Hash] interface {
	// Hash must return cryptographic hash of the transaction.
	// Transactions which have equal hashes are considered equal.
	Hash() H
}
