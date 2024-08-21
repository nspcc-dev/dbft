package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"math"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/internal/merkle"
)

type amevBlock struct {
	base

	transactions []dbft.Transaction[crypto.Uint256]
	signature    []byte
	hash         *crypto.Uint256
}

// NewAMEVBlock returns new block based on PreBlock and additional Commit-level data
// collected from M consensus nodes.
func NewAMEVBlock(pre dbft.PreBlock[crypto.Uint256], cnData [][]byte, m int) dbft.Block[crypto.Uint256] {
	preB := pre.(*preBlock)
	res := new(amevBlock)
	res.base = preB.base

	// Based on the provided cnData we'll add one more transaction to the resulting block.
	// Some artificial rules of new tx creation are invented here, but in Neo X there will
	// be well-defined custom rules for Envelope transactions.
	var sum uint32
	for i := range m {
		sum += binary.BigEndian.Uint32(cnData[i])
	}
	tx := Tx64(math.MaxInt64 - int64(sum))
	res.transactions = append(preB.initialTransactions, &tx)

	// Rebuild Merkle root for the new set of transations.
	txHashes := make([]crypto.Uint256, len(res.transactions))
	for i := range txHashes {
		txHashes[i] = res.transactions[i].Hash()
	}
	mt := merkle.NewMerkleTree(txHashes...)
	res.base.MerkleRoot = mt.Root().Hash

	return res
}

// PrevHash implements Block interface.
func (b *amevBlock) PrevHash() crypto.Uint256 {
	return b.base.PrevHash
}

// Index implements Block interface.
func (b *amevBlock) Index() uint32 {
	return b.base.Index
}

// MerkleRoot implements Block interface.
func (b *amevBlock) MerkleRoot() crypto.Uint256 {
	return b.base.MerkleRoot
}

// Transactions implements Block interface.
func (b *amevBlock) Transactions() []dbft.Transaction[crypto.Uint256] {
	return b.transactions
}

// SetTransactions implements Block interface. This method is special since it's
// left for dBFT 2.0 compatibility and transactions from this method must not be
// reused to fill final Block's transactions.
func (b *amevBlock) SetTransactions(_ []dbft.Transaction[crypto.Uint256]) {
}

// Signature implements Block interface.
func (b *amevBlock) Signature() []byte {
	return b.signature
}

// GetHashData returns data for hashing and signing.
// It must be an injection of the set of blocks to the set
// of byte slices, i.e:
// 1. It must have only one valid result for one block.
// 2. Two different blocks must have different hash data.
func (b *amevBlock) GetHashData() []byte {
	buf := bytes.Buffer{}
	w := gob.NewEncoder(&buf)
	_ = b.EncodeBinary(w)

	return buf.Bytes()
}

// Sign implements Block interface.
func (b *amevBlock) Sign(key dbft.PrivateKey) error {
	data := b.GetHashData()

	sign, err := key.Sign(data)
	if err != nil {
		return err
	}

	b.signature = sign

	return nil
}

// Verify implements Block interface.
func (b *amevBlock) Verify(pub dbft.PublicKey, sign []byte) error {
	data := b.GetHashData()
	return pub.(*crypto.ECDSAPub).Verify(data, sign)
}

// Hash implements Block interface.
func (b *amevBlock) Hash() (h crypto.Uint256) {
	if b.hash != nil {
		return *b.hash
	} else if b.transactions == nil {
		return
	}

	hash := crypto.Hash256(b.GetHashData())
	b.hash = &hash

	return hash
}
