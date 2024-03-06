package dbft

// Block is a generic interface for a block used by dbft.
type Block[H Hash] interface {
	// Hash returns block hash.
	Hash() H
	// PrevHash returns previous block hash.
	PrevHash() H
	// MerkleRoot returns a merkle root of the transaction hashes.
	MerkleRoot() H
	// Index returns block index.
	Index() uint32

	// Signature returns block's signature.
	Signature() []byte
	// Sign signs block and sets it's signature.
	Sign(key PrivateKey) error
	// Verify checks if signature is correct.
	Verify(key PublicKey, sign []byte) error

	// Transactions returns block's transaction list.
	Transactions() []Transaction[H]
	// SetTransactions sets block's transaction list.
	SetTransactions([]Transaction[H])
}
