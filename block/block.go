package block

import (
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/merkle"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

type (
	// base is a structure containing all
	// hashable and signable fields of the block.
	base struct {
		PrimaryIndex  byte
		Index         uint32
		Timestamp     uint32
		Version       uint32
		MerkleRoot    util.Uint256
		PrevHash      util.Uint256
		NextConsensus util.Uint160
	}

	// Block is a generic interface for a block used by dbft.
	Block interface {
		// Hash returns block hash.
		Hash() util.Uint256

		Version() uint32
		// PrevHash returns previous block hash.
		PrevHash() util.Uint256
		// MerkleRoot returns a merkle root of the transaction hashes.
		MerkleRoot() util.Uint256
		// Timestamp returns block's proposal timestamp.
		Timestamp() uint64
		// Index returns block index.
		Index() uint32
		// ConsensusData is a primary node index at the current round.
		PrimaryIndex() byte
		// NextConsensus returns hash of the validators of the next block.
		NextConsensus() util.Uint160

		// Signature returns block's signature.
		Signature() []byte
		// Sign signs block and sets it's signature.
		Sign(key crypto.PrivateKey) error
		// Verify checks if signature is correct.
		Verify(key crypto.PublicKey, sign []byte) error

		// Transactions returns block's transaction list.
		Transactions() []Transaction
		// SetTransaction sets block's transaction list.
		SetTransactions([]Transaction)
	}

	neoBlock struct {
		base

		transactions []Transaction
		signature    []byte
		hash         *util.Uint256
	}
)

// Version implements Block interface.
func (b neoBlock) Version() uint32 {
	return b.base.Version
}

// PrevHash implements Block interface.
func (b *neoBlock) PrevHash() util.Uint256 {
	return b.base.PrevHash
}

// Timestamp implements Block interface.
func (b *neoBlock) Timestamp() uint64 {
	return uint64(b.base.Timestamp) * 1000000000
}

// Index implements Block interface.
func (b *neoBlock) Index() uint32 {
	return b.base.Index
}

// NextConsensus implements Block interface.
func (b *neoBlock) NextConsensus() util.Uint160 {
	return b.base.NextConsensus
}

// MerkleRoot implements Block interface.
func (b *neoBlock) MerkleRoot() util.Uint256 {
	return b.base.MerkleRoot
}

// ConsensusData implements Block interface.
func (b *neoBlock) PrimaryIndex() byte {
	return b.base.PrimaryIndex
}

// Transactions implements Block interface.
func (b *neoBlock) Transactions() []Transaction {
	return b.transactions
}

// SetTransactions implements Block interface.
func (b *neoBlock) SetTransactions(txx []Transaction) {
	b.transactions = txx
}

// NewBlock returns new block.
func NewBlock(timestamp uint64, index uint32, nextConsensus util.Uint160, prevHash util.Uint256, version uint32, primaryIndex byte, txHashes []util.Uint256) Block {
	block := new(neoBlock)
	block.base.Timestamp = uint32(timestamp / 1000000000)
	block.base.Index = index
	block.base.NextConsensus = nextConsensus
	block.base.PrevHash = prevHash
	block.base.Version = version
	block.base.PrimaryIndex = primaryIndex

	if len(txHashes) != 0 {
		mt := merkle.NewMerkleTree(txHashes...)
		block.base.MerkleRoot = mt.Root().Hash
	}
	return block
}

// Signature implements Block interface.
func (b *neoBlock) Signature() []byte {
	return b.signature
}

// GetHashData returns data for hashing and signing.
// It must be an injection of the set of blocks to the set
// of byte slices, i.e:
// 1. It must have only one valid result for one block.
// 2. Two different blocks must have different hash data.
func (b *neoBlock) GetHashData() []byte {
	w := io.NewBufBinWriter()
	b.EncodeBinary(w.BinWriter)

	return w.Bytes()
}

// Sign implements Block interface.
func (b *neoBlock) Sign(key crypto.PrivateKey) error {
	data := b.GetHashData()

	sign, err := key.Sign(data)
	if err != nil {
		return err
	}

	b.signature = sign

	return nil
}

// Verify implements Block interface.
func (b *neoBlock) Verify(pub crypto.PublicKey, sign []byte) error {
	data := b.GetHashData()
	return pub.Verify(data, sign)
}

// Hash implements Block interface.
func (b *neoBlock) Hash() (h util.Uint256) {
	if b.hash != nil {
		return *b.hash
	} else if b.transactions == nil {
		return
	}

	hash := crypto.Hash256(b.GetHashData())
	b.hash = &hash

	return hash
}

// EncodeBinary implements io.Serializable interface.
func (b base) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(b.Version)
	w.WriteBytes(b.PrevHash[:])
	w.WriteBytes(b.MerkleRoot[:])
	w.WriteU32LE(b.Timestamp)
	w.WriteU32LE(b.Index)
	w.WriteB(b.PrimaryIndex)
	w.WriteBytes(b.NextConsensus[:])
}

// DecodeBinary implements io.Serializable interface.
func (b *base) DecodeBinary(r *io.BinReader) {
	b.Version = r.ReadU32LE()
	r.ReadBytes(b.PrevHash[:])
	r.ReadBytes(b.MerkleRoot[:])
	b.Timestamp = r.ReadU32LE()
	b.Index = r.ReadU32LE()
	b.PrimaryIndex = r.ReadB()
	r.ReadBytes(b.NextConsensus[:])
}
