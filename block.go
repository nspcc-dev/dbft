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
	// SetTransactions sets block's transaction list. For anti-MEV extension
	// transactions provided via this call are taken directly from PreBlock level
	// and thus, may be out-of-date. Thus, with anti-MEV extension enabled it's
	// suggested to use this method as a Block finalizer since it will be called
	// right before the block approval. Do not rely on this with anti-MEV extension
	// disabled.
	SetTransactions([]Transaction[H])
}
