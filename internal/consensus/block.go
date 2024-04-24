package consensus

import (
	"bytes"
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/internal/merkle"
)

type (
	// base is a structure containing all
	// hashable and signable fields of the block.
	base struct {
		ConsensusData uint64
		Index         uint32
		Timestamp     uint32
		Version       uint32
		MerkleRoot    crypto.Uint256
		PrevHash      crypto.Uint256
		NextConsensus crypto.Uint160
	}

	neoBlock struct {
		base

		transactions []dbft.Transaction[crypto.Uint256]
		signature    []byte
		hash         *crypto.Uint256
	}
)

// PrevHash implements Block interface.
func (b *neoBlock) PrevHash() crypto.Uint256 {
	return b.base.PrevHash
}

// Index implements Block interface.
func (b *neoBlock) Index() uint32 {
	return b.base.Index
}

// MerkleRoot implements Block interface.
func (b *neoBlock) MerkleRoot() crypto.Uint256 {
	return b.base.MerkleRoot
}

// Transactions implements Block interface.
func (b *neoBlock) Transactions() []dbft.Transaction[crypto.Uint256] {
	return b.transactions
}

// SetTransactions implements Block interface.
func (b *neoBlock) SetTransactions(txx []dbft.Transaction[crypto.Uint256]) {
	b.transactions = txx
}

// NewBlock returns new block.
func NewBlock(timestamp uint64, index uint32, prevHash crypto.Uint256, nonce uint64, txHashes []crypto.Uint256) dbft.Block[crypto.Uint256] {
	block := new(neoBlock)
	block.base.Timestamp = uint32(timestamp / 1000000000)
	block.base.Index = index

	// NextConsensus and Version information is not provided by dBFT context,
	// these are implementation-specific fields, and thus, should be managed outside the
	// dBFT library. For simulation simplicity, let's assume that these fields are filled
	// by every CN separately and is not verified.
	block.base.NextConsensus = crypto.Uint160{1, 2, 3}
	block.base.Version = 0

	block.base.PrevHash = prevHash
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
func (b *neoBlock) Sign(key dbft.PrivateKey) error {
	data := b.GetHashData()

	sign, err := key.Sign(data)
	if err != nil {
		return err
	}

	b.signature = sign

	return nil
}

// Verify implements Block interface.
func (b *neoBlock) Verify(pub dbft.PublicKey, sign []byte) error {
	data := b.GetHashData()
	return pub.(*crypto.ECDSAPub).Verify(data, sign)
}

// Hash implements Block interface.
func (b *neoBlock) Hash() (h crypto.Uint256) {
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
