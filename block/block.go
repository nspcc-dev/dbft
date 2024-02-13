package block

import (
	"bytes"
	"encoding/gob"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/merkle"
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
	Block[H crypto.Hash, A crypto.Address] interface {
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
		// NextConsensus returns hash of the validators of the next block.
		NextConsensus() A

		// Signature returns block's signature.
		Signature() []byte
		// Sign signs block and sets it's signature.
		Sign(key crypto.PrivateKey) error
		// Verify checks if signature is correct.
		Verify(key crypto.PublicKey, sign []byte) error

		// Transactions returns block's transaction list.
		Transactions() []Transaction[H]
		// SetTransactions sets block's transaction list.
		SetTransactions([]Transaction[H])
	}

	neoBlock struct {
		base

		consensusData uint64
		transactions  []Transaction[util.Uint256]
		signature     []byte
		hash          *util.Uint256
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
func (b *neoBlock) ConsensusData() uint64 {
	return b.consensusData
}

// Transactions implements Block interface.
func (b *neoBlock) Transactions() []Transaction[util.Uint256] {
	return b.transactions
}

// SetTransactions implements Block interface.
func (b *neoBlock) SetTransactions(txx []Transaction[util.Uint256]) {
	b.transactions = txx
}

// NewBlock returns new block.
func NewBlock(timestamp uint64, index uint32, nextConsensus util.Uint160, prevHash util.Uint256, version uint32, nonce uint64, txHashes []util.Uint256) Block[util.Uint256, util.Uint160] {
	block := new(neoBlock)
	block.base.Timestamp = uint32(timestamp / 1000000000)
	block.base.Index = index
	block.base.NextConsensus = nextConsensus
	block.base.PrevHash = prevHash
	block.base.Version = version
	block.base.ConsensusData = nonce

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
	buf := bytes.Buffer{}
	w := gob.NewEncoder(&buf)
	_ = b.EncodeBinary(w)

	return buf.Bytes()
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

// EncodeBinary implements Serializable interface.
func (b base) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(b)
}

// DecodeBinary implements Serializable interface.
func (b *base) DecodeBinary(r *gob.Decoder) error {
	return r.Decode(b)
}
