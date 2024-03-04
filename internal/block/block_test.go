package block

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"testing"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeoBlock_Setters(t *testing.T) {
	b := new(neoBlock)

	require.Equal(t, crypto.Uint256{}, b.Hash())

	txs := []dbft.Transaction[crypto.Uint256]{testTx(1), testTx(2)}
	b.SetTransactions(txs)
	assert.Equal(t, txs, b.Transactions())

	b.consensusData = 123
	assert.EqualValues(t, 123, b.ConsensusData())

	b.base.Version = 42
	assert.EqualValues(t, 42, b.Version())

	b.base.NextConsensus = crypto.Uint160{1}
	assert.Equal(t, crypto.Uint160{1}, b.NextConsensus())

	b.base.PrevHash = crypto.Uint256{3, 7}
	assert.Equal(t, crypto.Uint256{3, 7}, b.PrevHash())

	b.base.MerkleRoot = crypto.Uint256{13}
	assert.Equal(t, crypto.Uint256{13}, b.MerkleRoot())

	b.base.Timestamp = 1234
	// 1234s -> 1234000000000ns
	assert.EqualValues(t, uint64(1234000000000), b.Timestamp())

	b.base.Index = 100
	assert.EqualValues(t, 100, b.Index())

	t.Run("marshal block", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := gob.NewEncoder(&buf)
		err := b.EncodeBinary(w)
		require.NoError(t, err)

		r := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
		newb := new(neoBlock)
		err = newb.DecodeBinary(r)
		require.NoError(t, err)
		require.Equal(t, b.base, newb.base)
	})

	t.Run("hash does not change after signature", func(t *testing.T) {
		priv, pub := crypto.Generate(rand.Reader)
		require.NotNil(t, priv)
		require.NotNil(t, pub)

		h := b.Hash()
		require.NoError(t, b.Sign(priv))
		require.NotEmpty(t, b.Signature())
		require.Equal(t, h, b.Hash())
		require.NoError(t, b.Verify(pub, b.Signature()))
	})

	t.Run("sign with invalid private key", func(t *testing.T) {
		require.Error(t, b.Sign(testKey{}))
	})
}

type testKey struct{}

func (t testKey) MarshalBinary() ([]byte, error) { return []byte{}, nil }
func (t testKey) UnmarshalBinary([]byte) error   { return nil }
func (t testKey) Sign([]byte) ([]byte, error) {
	return nil, errors.New("can't sign")
}

type testTx uint64

func (tx testTx) Hash() (h crypto.Uint256) {
	binary.LittleEndian.PutUint64(h[:], uint64(tx))
	return
}