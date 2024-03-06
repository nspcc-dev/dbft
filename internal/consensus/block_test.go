package consensus

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

	b.base.PrevHash = crypto.Uint256{3, 7}
	assert.Equal(t, crypto.Uint256{3, 7}, b.PrevHash())

	b.base.MerkleRoot = crypto.Uint256{13}
	assert.Equal(t, crypto.Uint256{13}, b.MerkleRoot())

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
