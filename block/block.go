package block

import (
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

type (
	// base is a structure containing all
	// hashable and signable fields of the block.
	base struct {
		ConsensusData uint64
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
		SetVersion(uint32)
		// PrevHash returns previous block hash.
		PrevHash() util.Uint256
		// SetPrevHash sets PrevHash.
		SetPrevHash(util.Uint256)
		// MerkleRoot returns a merkle root of the transaction hashes.
		MerkleRoot() util.Uint256
		// SetMerkleRoot sets merkle tree's root.
		SetMerkleRoot(util.Uint256)
		// Timestamp returns block's proposal timestamp.
		Timestamp() uint32
		// SetTimestamp sets block timestamp.
		SetTimestamp(uint32)
		// Index returns block index.
		Index() uint32
		// SetIndex sets block index.
		SetIndex(uint32)
		// ConsensusData is a random nonce.
		ConsensusData() uint64
		// SetConsensusData sets consensus data.
		SetConsensusData(uint64)
		// NextConsensus returns hash of the validators of the next block.
		NextConsensus() util.Uint160
		// SetNextConsensus sets NextConsensus field.
		SetNextConsensus(util.Uint160)

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

		consensusData uint64
		transactions  []Transaction
		signature     []byte
		hash          *util.Uint256
	}
)

// Version implements Block interface.
func (b neoBlock) Version() uint32 {
	return b.base.Version
}

// SetVersion implements Block interface.
func (b *neoBlock) SetVersion(v uint32) {
	b.base.Version = v
}

// PrevHash implements Block interface.
func (b *neoBlock) PrevHash() util.Uint256 {
	return b.base.PrevHash
}

// SetPrevHash implements Block interface.
func (b *neoBlock) SetPrevHash(h util.Uint256) {
	b.base.PrevHash = h
}

// SetMerkleRoot implements Block interface.
func (b *neoBlock) SetMerkleRoot(r util.Uint256) {
	b.base.MerkleRoot = r
}

// Timestamp implements Block interface.
func (b *neoBlock) Timestamp() uint32 {
	return b.base.Timestamp
}

// SetTimestamp implements Block interface.
func (b *neoBlock) SetTimestamp(ts uint32) {
	b.base.Timestamp = ts
}

// Index implements Block interface.
func (b *neoBlock) Index() uint32 {
	return b.base.Index
}

// SetIndex implements Block interface.
func (b *neoBlock) SetIndex(i uint32) {
	b.base.Index = i
}

// NextConsensus implements Block interface.
func (b *neoBlock) NextConsensus() util.Uint160 {
	return b.base.NextConsensus
}

// SetNextConsensus implements Block interface.
func (b *neoBlock) SetNextConsensus(h util.Uint160) {
	b.base.NextConsensus = h
}

// MerkleRoot implements Block interface.
func (b *neoBlock) MerkleRoot() util.Uint256 {
	return b.base.MerkleRoot
}

// ConsensusData implements Block interface.
func (b *neoBlock) ConsensusData() uint64 {
	return b.consensusData
}

// SetConsensusData implements Block interface.
func (b *neoBlock) SetConsensusData(cd uint64) {
	b.consensusData = cd
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
func NewBlock() Block {
	return new(neoBlock)
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
	w.WriteU64LE(b.ConsensusData)
	w.WriteBytes(b.NextConsensus[:])
}

// DecodeBinary implements io.Serializable interface.
func (b *base) DecodeBinary(r *io.BinReader) {
	b.Version = r.ReadU32LE()
	r.ReadBytes(b.PrevHash[:])
	r.ReadBytes(b.MerkleRoot[:])
	b.Timestamp = r.ReadU32LE()
	b.Index = r.ReadU32LE()
	b.ConsensusData = r.ReadU64LE()
	r.ReadBytes(b.NextConsensus[:])
}
