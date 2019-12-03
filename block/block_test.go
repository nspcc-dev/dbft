package block

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/CityOfZion/neo-go/pkg/util"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeoBlock_Setters(t *testing.T) {
	b := new(neoBlock)

	require.Equal(t, util.Uint256{}, b.Hash())

	txs := []Transaction{testTx(1), testTx(2)}
	b.SetTransactions(txs)
	assert.Equal(t, txs, b.Transactions())

	b.SetConsensusData(123)
	assert.EqualValues(t, 123, b.ConsensusData())

	b.SetVersion(42)
	assert.EqualValues(t, 42, b.Version())

	b.SetNextConsensus(util.Uint160{1})
	assert.Equal(t, util.Uint160{1}, b.NextConsensus())

	b.SetPrevHash(util.Uint256{3, 7})
	assert.Equal(t, util.Uint256{3, 7}, b.PrevHash())

	b.SetMerkleRoot(util.Uint256{13})
	assert.Equal(t, util.Uint256{13}, b.MerkleRoot())

	b.SetTimestamp(12345)
	assert.EqualValues(t, 12345, b.Timestamp())

	b.SetIndex(100)
	assert.EqualValues(t, 100, b.Index())

	t.Run("marshal block", func(t *testing.T) {
		w := io.NewBufBinWriter()
		b.EncodeBinary(w.BinWriter)
		require.NoError(t, w.Err)

		r := io.NewBinReaderFromBuf(w.Bytes())
		newb := new(neoBlock)
		newb.DecodeBinary(r)
		require.NoError(t, r.Err)
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

func (t testKey) MarshalBinary() (data []byte, err error) { return []byte{}, nil }
func (t testKey) UnmarshalBinary(data []byte) error       { return nil }
func (t testKey) Sign(msg []byte) (sig []byte, err error) {
	return nil, errors.New("can't sign")
}

type testTx uint64

func (tx testTx) Hash() (h util.Uint256) {
	binary.LittleEndian.PutUint64(h[:], uint64(tx))
	return
}
