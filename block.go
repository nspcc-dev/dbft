package dbft

// Block is a generic interface for a block used by dbft.
type Block[H Hash] interface {
	// Hash returns block hash.
	Hash() H

	Version() uint32
	// PrevHash returns previous block hash.
	PrevHash() H
	// MerkleRoot returns a merkle root of the transaction hashes.
	MerkleRoot() H
	// Timestamp returns block's proposal timestamp.
	Timestamp() uint64
	// Index returns block index.
	Index() uint32
	// ConsensusData is a random nonce.
	ConsensusData() uint64

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
