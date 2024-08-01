package consensus

import (
	"encoding/binary"
	"errors"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/internal/merkle"
)

type preBlock struct {
	base

	// A magic number CN nodes should exchange during Commit phase
	// and used to construct the final list of transactions for amevBlock.
	data uint32

	initialTransactions []dbft.Transaction[crypto.Uint256]
}

var _ dbft.PreBlock[crypto.Uint256] = new(preBlock)

// NewPreBlock returns new preBlock.
func NewPreBlock(timestamp uint64, index uint32, prevHash crypto.Uint256, nonce uint64, txHashes []crypto.Uint256) dbft.PreBlock[crypto.Uint256] {
	pre := new(preBlock)
	pre.base.Timestamp = uint32(timestamp / 1000000000)
	pre.base.Index = index

	// NextConsensus and Version information is not provided by dBFT context,
	// these are implementation-specific fields, and thus, should be managed outside the
	// dBFT library. For simulation simplicity, let's assume that these fields are filled
	// by every CN separately and is not verified.
	pre.base.NextConsensus = crypto.Uint160{1, 2, 3}
	pre.base.Version = 0

	pre.base.PrevHash = prevHash
	pre.base.ConsensusData = nonce

	// Canary default value.
	pre.data = 0xff

	if len(txHashes) != 0 {
		mt := merkle.NewMerkleTree(txHashes...)
		pre.base.MerkleRoot = mt.Root().Hash
	}
	return pre
}

func (pre *preBlock) Data() []byte {
	var res = make([]byte, 4)
	binary.BigEndian.PutUint32(res, pre.data)
	return res
}

func (pre *preBlock) SetData(_ dbft.PrivateKey) error {
	// Just an artificial rule for data construction, it can be anything, and in Neo X
	// it will be decrypted transactions fragments.
	pre.data = pre.base.Index
	return nil
}

func (pre *preBlock) Verify(_ dbft.PublicKey, data []byte) error {
	if len(data) != 4 {
		return errors.New("invalid data len")
	}
	if binary.BigEndian.Uint32(data) != pre.base.Index { // Just an artificial verification rule, and for NeoX it should be decrypted transactions fragments verification.
		return errors.New("invalid data")
	}
	return nil
}

func (pre *preBlock) Transactions() []dbft.Transaction[crypto.Uint256] {
	return pre.initialTransactions
}

func (pre *preBlock) SetTransactions(txs []dbft.Transaction[crypto.Uint256]) {
	pre.initialTransactions = txs
}
